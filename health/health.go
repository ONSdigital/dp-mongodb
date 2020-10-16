package health

import (
	"context"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/log"
	mgo "github.com/globalsign/mgo"
)

// ServiceName mongodb
const ServiceName = "mongodb"

var (
	// HealthyMessage is the message that will be used in healthcheck when mongo is Healthy and all the collections exist
	HealthyMessage = "mongodb is OK and all expected collections exist"
)

// Healthcheck health check function
type Healthcheck = func(context.Context) (string, error)

// CheckMongoClient is an implementation of the mongo client with a healthcheck
type CheckMongoClient struct {
	Client      Client
	Healthcheck Healthcheck
}

type (
	//Database a list of mongo types
	Database string
	//Collection a list of mongo types
	Collection string
)

//go:generate moq -out health_moq_test.go . sessioner
type sessioner interface {
	DB(name string) *mgo.Database
	Copy() *mgo.Session
	Close() *mgo.Session
	Ping() *mgo.Session
}

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo              sessioner
	serviceName        string
	databaseCollection map[Database][]Collection
}

// NewClient returns a new health check client using the given service
func NewClient(db sessioner, clientDatabaseCollection map[Database][]Collection) *Client {
	return &Client{
		mongo:              db,
		serviceName:        ServiceName,
		databaseCollection: clientDatabaseCollection,
	}
}

func (m *Client) checkCollections(ctx context.Context) (err error) {

	for database, collections := range m.databaseCollection {

		logData := log.Data{"Database": string(database)}
		collectionsInDb, err := m.mongo.DB(string(database)).CollectionNames()
		if err != nil {
			log.Event(ctx, "Failed to connect to mongoDB to get the collections", log.ERROR, logData, log.Error(err))
		}

		for _, collection := range collections {
			logData := log.Data{"Database": string(database), "Collection": string(collection)}
			for _, collectionInDb := range collectionsInDb {
				if string(collection) == collectionInDb {
					break
				} else {
					log.Event(ctx, "Collection does not exist in the database", log.ERROR, logData, log.Error(err))
					return err
				}
			}
		}
	}
	return nil
}

// Healthcheck calls service to check its health status
func (m *Client) Healthcheck(ctx context.Context) (res string, err error) {
	s := m.mongo.Copy()
	defer s.Close()
	res = m.serviceName
	err = s.Ping()
	if err != nil {
		log.Event(ctx, "Ping mongo", log.ERROR, log.Error(err))
	}

	m.checkCollections(ctx)
	if err != nil {
		log.Event(ctx, "Error checking collections in mongo", log.ERROR, log.Error(err))
	}

	return
}

// Checker calls an api health endpoint and  updates the provided CheckState accordingly
func (c *CheckMongoClient) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	_, err := c.Healthcheck(ctx)
	if err != nil {
		state.Update(healthcheck.StatusCritical, err.Error(), 0)
		return nil
	}
	state.Update(healthcheck.StatusOK, HealthyMessage, 0)
	return nil
}
