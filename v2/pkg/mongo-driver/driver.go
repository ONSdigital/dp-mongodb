package mongo_driver

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"

	"log"
	"time"
)

const (
	connectionStringTemplate = "mongodb://%s:%s@%s/sample-database?ssl=true&replicaSet=rs0"
)

type Driver interface {
	Open()
}

type MongoDriver struct {
	caFilePath              string
	connectTimeoutInSeconds time.Duration
	queryTimeoutInSeconds   time.Duration

	username        string
	password        string
	clusterEndpoint string
	database        string
	collection      string
}

func (m *MongoDriver) getConnectionURI() string {
	return fmt.Sprintf(connectionStringTemplate, m.username, m.password, m.clusterEndpoint)
}

func (m *MongoDriver) Open() (*MongoSession, error) {
	var tlsConfig *tls.Config
	var err error
	if len(m.caFilePath) > 0 {
		tlsConfig, err = getCustomTLSConfig(m.caFilePath)
		if err != nil {
			errMessage := fmt.Sprintf("Failed getting TLS configuration: %v", err)
			log.Fatalf(errMessage)
			return nil, errors.New(errMessage)
		}
	}

	mongoClientOptions := options.Client().
		ApplyURI(m.getConnectionURI()).
		SetTLSConfig(tlsConfig).
		SetReadPreference(readpref.PrimaryPreferred()).
		// For ensuring strong consistency
		SetReadConcern(readconcern.Majority()).
		SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

	var client *mongo.Client
	client, err = mongo.NewClient(mongoClientOptions)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to create client: %v", err)
		log.Fatalf(errMessage)
		return nil, errors.New(errMessage)
	}

	connectionCtx, cancel := context.WithTimeout(context.Background(), m.connectTimeoutInSeconds*time.Second)
	defer cancel()

	err = client.Connect(connectionCtx)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to connect to cluster: %v", err)
		log.Fatalf(errMessage)
		return nil, errors.New(errMessage)
	}

	// Force a connection to verify our connection string
	err = client.Ping(connectionCtx, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to ping cluster: %v", err)
		log.Fatalf(errMessage)
		return nil, errors.New(errMessage)
	}

	return NewMongoSession(client, m.database, m.collection), nil
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
