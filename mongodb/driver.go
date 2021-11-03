package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/ONSdigital/log.go/v2/log"
)

const (
	connectionStringTemplate             = "mongodb://%s:%s@%s/%s"
	connectionStringTemplateWithoutCreds = "mongodb://%s/%s"
	int64Size                            = 64
)

// TLSConnectionConfig supplies the options for setting up a TLS based connection to the Mongo DB server
// If the Mongo server certificate is to be validated (a major security breach not doing so), the VerifyCert
// should be true, and the chain of CA certificates for the validation must be supplied -  in a file specified
// by the absolute path given in the CACertChain attribute.
// If the connection to the server is being made with an IP address, or via an SSH proxy
// (such as with `dp ssh develop publishing 1 -p local-port:remote-host:remote-port`)
// the real hostname should be supplied in the RealHostnameForSSH attribute. The real hostname is the
// name of the server as attested by the server's x509 certificate. So in the above example of a connection via ssh
// this would be the value of `remotehost`
type TLSConnectionConfig struct {
	IsSSL              bool
	VerifyCert         bool
	CACertChain        string
	RealHostnameForSSH string
}

var (
	NoCACertChain      = errors.New("no CA certificate chain supplied, or chain cannot be read")
	InvalidCACertChain = errors.New("cannot parse CA certificate chain - invalid or corrupt")
)

func (m TLSConnectionConfig) GetTLSConfig() (*tls.Config, error) {
	if !m.IsSSL {
		return nil, nil
	}

	if !m.VerifyCert {
		return &tls.Config{InsecureSkipVerify: true}, nil
	}

	if m.CACertChain == "" {
		return nil, NoCACertChain
	}

	certChain, e := os.ReadFile(m.CACertChain)
	if e != nil {
		return nil, NoCACertChain
	}

	tlsConfig := &tls.Config{}
	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certChain)
	if !ok {
		return nil, InvalidCACertChain
	}

	if m.RealHostnameForSSH != "" {
		tlsConfig.ServerName = m.RealHostnameForSSH
	}

	return tlsConfig, nil
}

type MongoConnectionConfig struct {
	Username                      string
	Password                      string
	ClusterEndpoint               string
	Database                      string
	Collection                    string
	ReplicaSet                    string
	IsStrongReadConcernEnabled    bool
	IsWriteConcernMajorityEnabled bool

	ConnectTimeoutInSeconds time.Duration
	QueryTimeoutInSeconds   time.Duration

	TLSConnectionConfig
}

func (m *MongoConnectionConfig) GetConnectionURI() string {
	var connectionString string

	if len(m.Password) > 0 && len(m.Username) > 0 {
		connectionString = fmt.Sprintf(connectionStringTemplate, m.Username, m.Password, m.ClusterEndpoint, m.Database)
	} else {
		connectionString = fmt.Sprintf(connectionStringTemplateWithoutCreds, m.ClusterEndpoint, m.Database)
	}

	return connectionString
}

func Open(m *MongoConnectionConfig) (*MongoConnection, error) {
	if strconv.IntSize < int64Size {
		return nil, errors.New("cannot use dp-mongodb library when default int size is less than 64 bits")
	}

	tlsConfig, err := m.GetTLSConfig()
	if err != nil {
		errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
		log.Error(context.Background(), errMessage, err)
		return nil, err
	}

	mongoClientOptions := options.Client().
		ApplyURI(m.GetConnectionURI()).
		SetTLSConfig(tlsConfig).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetRetryWrites(false)

	if m.ReplicaSet != "" {
		mongoClientOptions = mongoClientOptions.SetReplicaSet(m.ReplicaSet)
	} else {
		mongoClientOptions = mongoClientOptions.SetDirect(true)
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
