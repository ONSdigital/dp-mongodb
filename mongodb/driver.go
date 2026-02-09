package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
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
	"github.com/hahnicity/go-wget"
)

const (
	connectionStringTemplateWithoutCreds = "mongodb://%s/%s"
	connectionStringTemplateAWS          = "mongodb://%s:%s@%s/migrations?tls=true&replicaSet=rs0&readpreference=%s"
	connectionStringTemplateStandard     = "mongodb://%s:%s@%s/%s"
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

func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)

	if err != nil {
		return tlsConfig, err
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}

type MongoDriverConfig struct {
	ConnectEKS      bool   `envconfig:"CONNECT_EKS" json:"-"`
	Username        string `envconfig:"MONGODB_USERNAME"    json:"-"`
	Password        string `envconfig:"MONGODB_PASSWORD"    json:"-"`
	ClusterEndpoint string `envconfig:"MONGODB_BIND_ADDR"   json:"-"`
	Database        string `envconfig:"MONGODB_DATABASE"`
	// Collections is a mapping from a collection's 'Well Known Name' to 'Actual Name'
	Collections                   map[string]string `envconfig:"MONGODB_COLLECTIONS"`
	ReplicaSet                    string            `envconfig:"MONGODB_REPLICA_SET"`
	DirectConnection              bool              `envconfig:"MONGODB_DIRECT_CONNECTION"`
	IsStrongReadConcernEnabled    bool              `envconfig:"MONGODB_ENABLE_READ_CONCERN"`
	IsWriteConcernMajorityEnabled bool              `envconfig:"MONGODB_ENABLE_WRITE_CONCERN"`

	ConnectTimeout time.Duration `envconfig:"MONGODB_CONNECT_TIMEOUT"`
	QueryTimeout   time.Duration `envconfig:"MONGODB_QUERY_TIMEOUT"`

	TLSConnectionConfig
}

func (m *MongoDriverConfig) ActualCollectionName(wellKnownName string) string {
	return m.Collections[wellKnownName]
}

//func getIAMCredentials(ctx context.Context) (username, password string, err error) {
//	// Load the Shared AWS Configuration (~/.aws/config)
//	cfg, err := config.LoadDefaultConfig(ctx)
//	if err != nil {
//		return "", "", err
//	}
//
//	creds, err := cfg.Credentials.Retrieve(ctx)
//	if err != nil {
//		return "", "", err
//	}
//
//	accessKeyID := creds.AccessKeyID
//	secretAccessKey := creds.SecretAccessKey
//	logLine := fmt.Sprintf("The value of username is %s and the password is %s", accessKeyID, secretAccessKey)
//	log.Info(ctx, logLine)
//	encodedSecretAccessKey := url.QueryEscape(creds.SecretAccessKey)
//	logLine = fmt.Sprintf("The encoded password is %s", encodedSecretAccessKey)
//	log.Info(ctx, logLine)
//
//	return creds.AccessKeyID, encodedSecretAccessKey, err
//}

func Open(ctx context.Context, m *MongoDriverConfig) (*MongoConnection, error) {
	if strconv.IntSize < int64Size {
		return nil, errors.New("cannot use dp-mongodb library when default int size is less than 64 bits")
	}

	var connectionString string
	var client *mongo.Client

	if !m.ConnectEKS {
		endpoint := m.ClusterEndpoint

		matches, err := regexp.MatchString(endpointRegex, endpoint)
		if err != nil {
			return nil, err
		}
		if !matches {
			return nil, fmt.Errorf("invalid mongodb address: %s", endpoint)
		}

		endpoint = strings.TrimPrefix(endpoint, "mongodb://")

		if len(m.Password) > 0 && len(m.Username) > 0 {
			connectionString = fmt.Sprintf(connectionStringTemplateStandard, m.Username, m.Password, endpoint, m.Database)
			logLine := fmt.Sprintf("The standard connection string is %s", connectionString)
			log.Info(ctx, logLine)
		} else {
			connectionString = fmt.Sprintf(connectionStringTemplateWithoutCreds, endpoint, m.Database)
			logLine := fmt.Sprintf("The connection string without credentials is %s", connectionString)
			log.Info(ctx, logLine)
		}

		if m.ReplicaSet != "" {
			connectionString += fmt.Sprintf("?replicaSet=%s", m.ReplicaSet)
			if m.DirectConnection {
				connectionString += "&directConnection=true"
			}
		} else {
			connectionString += "?directConnection=true"
		}

		tlsConfig, err := m.GetTLSConfig()
		if err != nil {
			errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
			log.Error(ctx, errMessage, err)
			return nil, err
		}

		mongoClientOptions := options.Client().
			ApplyURI(connectionString).
			SetTLSConfig(tlsConfig).
			SetRetryWrites(false)

		if m.IsStrongReadConcernEnabled {
			// For ensuring strong consistency
			mongoClientOptions = mongoClientOptions.SetReadPreference(readpref.Primary())
			// The following is needed for MongoDB but has no effect for DocumentDB
			mongoClientOptions = mongoClientOptions.SetReadConcern(readconcern.Majority())
		} else {
			mongoClientOptions = mongoClientOptions.SetReadPreference(readpref.SecondaryPreferred())
		}

		if m.IsWriteConcernMajorityEnabled {
			mongoClientOptions = mongoClientOptions.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))
		} else {
			mongoClientOptions = mongoClientOptions.SetWriteConcern(writeconcern.New(writeconcern.W(1)))
		}

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
		fmt.Println("Connected to DocumentDB using legacy settings!")
	} else {
		// Path to the AWS CA file
		caFilePath := "global-bundle.pem"
		wget.Wget("https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem", caFilePath)

		// Timeout operations after N seconds
		//connectTimeout := 5
		readPreference := "secondaryPreferred"
		connectionString := fmt.Sprintf(connectionStringTemplateAWS, m.Username, m.Password, m.ClusterEndpoint, readPreference)
		logLine := fmt.Sprintf("The connection string from EKS is %s", connectionString)
		log.Info(ctx, logLine)

		tlsConfig, err := getCustomTLSConfig(caFilePath)
		if err != nil {
			log.Fatal(ctx, "failed getting TLS configuration", err)
		}

		client, err := mongo.NewClient(options.Client().ApplyURI(connectionString).SetTLSConfig(tlsConfig))
		if err != nil {
			log.Fatal(ctx, "failed to create client", err)
		}

		var timeout time.Duration
		timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err = client.Connect(ctx)
		if err != nil {
			log.Fatal(ctx, "failed to connect to cluster", err)
		}

		// Force a connection to verify our connection string
		err = client.Ping(ctx, nil)
		if err != nil {
			log.Fatal(ctx, "failed to ping cluster", err)
		}

		fmt.Println("Connected to DocumentDB using EKS settings!")
	}
	return NewMongoConnection(client, m.Database), nil
}
