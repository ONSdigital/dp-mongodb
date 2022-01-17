package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	endpointRegex                        = "^(mongodb://)?[^:/]+(:\\d+)?$"
)

// TLSConnectionConfig supplies the options for setting up a TLS based connection to the Mongo DB server
// If the Mongo server certificate is to be validated (a major security breach not doing so), the VerifyCert
// should be true, and the chain of CA certificates for the validation must be supplied
// If the connection to the server is being made with an IP address, or via an SSH proxy
// (such as with `dp ssh develop publishing 1 -p local-port:remote-host:remote-port`)
// the real hostname should be supplied in the RealHostnameForSSH attribute. The real hostname is the
// name of the server as attested by the server's x509 certificate. So in the above example of a connection via ssh
// this would be the value of `remotehost`
type TLSConnectionConfig struct {
	IsSSL              bool   `envconfig:"MONGODB_IS_SSL"`
	VerifyCert         bool   `envconfig:"MONGODB_VERIFY_CERT"`
	CACertChain        string `envconfig:"MONGODB_CERT_CHAIN"`
	RealHostnameForSSH string `envconfig:"MONGODB_REAL_HOSTNAME"`
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

	tlsConfig := &tls.Config{}
	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(m.CACertChain))
	if !ok {
		return nil, InvalidCACertChain
	}

	if m.RealHostnameForSSH != "" {
		tlsConfig.ServerName = m.RealHostnameForSSH
	}

	return tlsConfig, nil
}

type MongoDriverConfig struct {
	Username        string `envconfig:"MONGODB_USERNAME"    json:"-"`
	Password        string `envconfig:"MONGODB_PASSWORD"    json:"-"`
	ClusterEndpoint string `envconfig:"MONGODB_BIND_ADDR"   json:"-"`
	Database        string `envconfig:"MONGODB_DATABASE"`
	// Collections is a mapping from a collection's 'Well Known Name' to 'Actual Name'
	Collections                   map[string]string `envconfig:"MONGODB_COLLECTIONS"`
	ReplicaSet                    string            `envconfig:"MONGODB_REPLICA_SET"`
	IsStrongReadConcernEnabled    bool              `envconfig:"MONGODB_ENABLE_READ_CONCERN"`
	IsWriteConcernMajorityEnabled bool              `envconfig:"MONGODB_ENABLE_WRITE_CONCERN"`

	ConnectTimeout time.Duration `envconfig:"MONGODB_CONNECT_TIMEOUT"`
	QueryTimeout   time.Duration `envconfig:"MONGODB_QUERY_TIMEOUT"`

	TLSConnectionConfig
}

func (m *MongoDriverConfig) ActualCollectionName(wellKnownName string) string {
	return m.Collections[wellKnownName]
}

func (m *MongoDriverConfig) GetConnectionURI() (string, error) {
	var connectionString string
	endpoint := m.ClusterEndpoint

	matches, err := regexp.MatchString(endpointRegex, endpoint)
	if err != nil {
		return "", err
	}
	if !matches {
		return "", fmt.Errorf("invalid mongodb address: %s", endpoint)
	}

	endpoint = strings.TrimPrefix(endpoint, "mongodb://")

	if len(m.Password) > 0 && len(m.Username) > 0 {
		connectionString = fmt.Sprintf(connectionStringTemplate, m.Username, m.Password, endpoint, m.Database)
	} else {
		connectionString = fmt.Sprintf(connectionStringTemplateWithoutCreds, endpoint, m.Database)
	}

	if m.ReplicaSet != "" {
		connectionString += fmt.Sprintf("?replicaSet=%s", m.ReplicaSet)
	} else {
		connectionString += "?directConnection=true"
	}

	return connectionString, nil
}

func Open(m *MongoDriverConfig) (*MongoConnection, error) {
	if strconv.IntSize < int64Size {
		return nil, errors.New("cannot use dp-mongodb library when default int size is less than 64 bits")
	}

	tlsConfig, err := m.GetTLSConfig()
	if err != nil {
		errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
		log.Error(context.Background(), errMessage, err)
		return nil, err
	}

	connectionUri, err := m.GetConnectionURI()
	if err != nil {
		return nil, err
	}

	mongoClientOptions := options.Client().
		ApplyURI(connectionUri).
		SetTLSConfig(tlsConfig).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetRetryWrites(false)

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

	connectionCtx, cancel := context.WithTimeout(context.Background(), m.ConnectTimeout)
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

	return NewMongoConnection(client, m.Database), nil
}
