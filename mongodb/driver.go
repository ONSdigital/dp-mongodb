package mongodb

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crypto/tls"
	"errors"
	"time"

	"github.com/ONSdigital/log.go/v2/log"
)

const (
	connectionStringTemplate             = "mongodb://%s:%s@%s/%s"
	connectionStringTemplateWithoutCreds = "mongodb://%s/%s"
	int64Size                            = 64
)

type MongoConnectionConfig struct {
	IsSSL                   bool
	ConnectTimeoutInSeconds time.Duration
	QueryTimeoutInSeconds   time.Duration

	Username                      string
	Password                      string
	ClusterEndpoint               string
	Database                      string
	Collection                    string
	ReplicaSet                    string
	IsStrongReadConcernEnabled    bool
	IsWriteConcernMajorityEnabled bool
}

func (m *MongoConnectionConfig) GetConnectionURI(isSSL bool) string {
	var connectionString string

	if len(m.Password) > 0 && len(m.Username) > 0 {
		connectionString = fmt.Sprintf(connectionStringTemplate, m.Username, m.Password, m.ClusterEndpoint, m.Database)
	} else {
		connectionString = fmt.Sprintf(connectionStringTemplateWithoutCreds, m.ClusterEndpoint, m.Database)
	}

	if isSSL {
		connectionString = strings.Join([]string{connectionString, "ssl=true"}, "?")
		connectionString = strings.Join([]string{connectionString, "connect=direct"}, "&")
		connectionString = strings.Join([]string{connectionString, "sslInsecure=true"}, "&")
	}

	return connectionString
}

func Open(m *MongoConnectionConfig) (*MongoConnection, error) {
	if strconv.IntSize < int64Size {
		return nil, errors.New("cannot use dp-mongodb library when default int size is less than 64 bits")
	}

	var tlsConfig *tls.Config
	var err error
	if m.IsSSL {
		tlsConfig, err = getCustomTLSConfig(true)
		if err != nil {
			errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
			log.Error(context.Background(), errMessage, err)
			return nil, errors.New(errMessage)
		}
	}

	uri := m.GetConnectionURI(m.IsSSL)
	mongoClientOptions := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsConfig).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetRetryWrites(false)

	if len(m.ReplicaSet) > 0 {
		mongoClientOptions = mongoClientOptions.SetReplicaSet(m.ReplicaSet)
	}

	if m.IsStrongReadConcernEnabled {
		// For ensuring strong consistency
		mongoClientOptions = mongoClientOptions.SetReadConcern(readconcern.Majority())
	}

	if m.IsWriteConcernMajorityEnabled {
		mongoClientOptions = mongoClientOptions.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
		// No support for retryable writes, retryable commit and retryable abort.
		//https://docs.aws.amazon.com/documentdb/latest/developerguide/transactions.html
		//https://docs.aws.amazon.com/documentdb/latest/developerguide/functional-differences.html#functional-differences.retryable-writes
	}

	var client *mongo.Client
	client, err = mongo.NewClient(mongoClientOptions)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to create client: %v", err)
		log.Error(context.Background(), errMessage, err)
		return nil, errors.New(errMessage)
	}

	connectionCtx, cancel := context.WithTimeout(context.Background(), m.ConnectTimeoutInSeconds*time.Second)
	defer cancel()

	err = client.Connect(connectionCtx)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to connect to cluster: %v", err)
		log.Error(context.Background(), errMessage, err)
		return nil, errors.New(errMessage)
	}

	// Force a connection to verify our connection string
	err = client.Ping(connectionCtx, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to ping cluster: %v", err)
		log.Error(context.Background(), errMessage, err)
		return nil, errors.New(errMessage)
	}

	return NewMongoConnection(client, m.Database, m.Collection), nil
}

func getCustomTLSConfig(skipCertVerification bool) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	if skipCertVerification {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig, nil
}
