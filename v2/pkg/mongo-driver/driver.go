package mongo_driver

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"time"

	"github.com/ONSdigital/log.go/log"
)

const (
	connectionStringTemplate = "mongodb://%s:%s@%s/%s"
)

type MongoConnectionConfig struct {
	CaFilePath              string
	ConnectTimeoutInSeconds time.Duration
	QueryTimeoutInSeconds   time.Duration

	Username             string
	Password             string
	ClusterEndpoint      string
	Database             string
	Collection           string
	replicaSet           string
	SkipCertVerification bool
}

func (m *MongoConnectionConfig) getConnectionURI(isSSL bool) string {
	connectionString := fmt.Sprintf(connectionStringTemplate, m.Username, m.Password, m.ClusterEndpoint, m.Database)
	if isSSL {
		connectionString = strings.Join([]string{connectionString, "ssl=true"}, "?")
	}
	return connectionString
}

func Open(m *MongoConnectionConfig) (*MongoConnection, error) {
	var tlsConfig *tls.Config
	var err error
	isSSL := len(m.CaFilePath) > 0
	if isSSL {
		tlsConfig, err = getCustomTLSConfig(m.CaFilePath, m.SkipCertVerification)
		if err != nil {
			errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
			log.Event(context.Background(), errMessage, log.ERROR, log.Error(err))
			return nil, errors.New(errMessage)
		}
	}

	uri := m.getConnectionURI(isSSL)
	fmt.Println(uri)
	mongoClientOptions := options.Client().
		ApplyURI(uri).
		SetTLSConfig(tlsConfig).
		SetReadPreference(readpref.PrimaryPreferred()).
		// For ensuring strong consistency
		SetReadConcern(readconcern.Majority()).
		SetWriteConcern(writeconcern.New(writeconcern.WMajority())).
		// No support for retryable writes, retryable commit and retryable abort.
		//https://docs.aws.amazon.com/documentdb/latest/developerguide/transactions.html
		//https://docs.aws.amazon.com/documentdb/latest/developerguide/functional-differences.html#functional-differences.retryable-writes
		SetRetryWrites(false)

	if len(m.replicaSet) > 0 {
		mongoClientOptions = mongoClientOptions.SetReplicaSet(m.replicaSet)
	}

	var client *mongo.Client
	client, err = mongo.NewClient(mongoClientOptions)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to create client: %v", err)
		log.Event(context.Background(), errMessage, log.ERROR, log.Error(err))
		return nil, errors.New(errMessage)
	}

	connectionCtx, cancel := context.WithTimeout(context.Background(), m.ConnectTimeoutInSeconds*time.Second)
	defer cancel()

	err = client.Connect(connectionCtx)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to connect to cluster: %v", err)
		log.Event(context.Background(), errMessage, log.ERROR, log.Error(err))
		return nil, errors.New(errMessage)
	}

	// Force a connection to verify our connection string
	err = client.Ping(connectionCtx, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to ping cluster: %v", err)
		log.Event(context.Background(), errMessage, log.ERROR, log.Error(err))
		return nil, errors.New(errMessage)
	}

	return NewMongoConnection(client, m.Database, m.Collection), nil
}

func getCustomTLSConfig(caFile string, skipCertVerification bool) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)

	if err != nil {
		return nil, err
	}

	if skipCertVerification {
		tlsConfig.InsecureSkipVerify = true
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}
