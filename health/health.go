package health

import (
	"context"
	"errors"

	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/v2/log"
)

const timeOutInSeconds = 5

var (
	// healthyMessage is the message that will be used in healthcheck when mongo is healthy
	healthyMessage = "mongodb is OK"
	// healthyCollectionsMessage is the message that will be appended in healthcheck when all the collections exist
	healthyCollectionsMessage = " and all expected collections exist"
)

// List of errors
var (
	errorCollectionDoesNotExist = errors.New("collection not found in database")
	errorWithMongoDBConnection  = errors.New("unable to connect with MongoDB")
)

// CheckMongoClient provides a healthcheck.Client implementation for health checking the service
type CheckMongoClient struct {
	client
	healthcheck      func(context.Context) error
	checkCollections func(context.Context) error
}

type (
	//Database a list of mongo types
	Database string
	//Collection a list of mongo types
	Collection string
)

// client provides a healthcheck.Client implementation for health checking the service
type client struct {
	mongoConnection    *mongoDriver.MongoConnection
	databaseCollection map[Database][]Collection
}

// NewClient returns a new health check client using the given service
func NewClient(mongoConnection *mongoDriver.MongoConnection) *CheckMongoClient {
	return NewClientWithCollections(mongoConnection, nil)
}

// NewClientWithCollections returns a new health check client containing the collections using the given service
func NewClientWithCollections(mongoConnection *mongoDriver.MongoConnection, clientDatabaseCollection map[Database][]Collection) *CheckMongoClient {
	c := client{
		mongoConnection:    mongoConnection,
		databaseCollection: clientDatabaseCollection,
	}
	return &CheckMongoClient{
		client:           c,
		healthcheck:      c.healthcheck,
		checkCollections: c.checkCollections,
	}
}

func (m *client) checkCollections(ctx context.Context) (err error) {
	for databaseToCheck, collectionsToCheck := range m.databaseCollection {

		logData := log.Data{"Database": string(databaseToCheck)}
		collectionsInDb, err := m.mongoConnection.ListCollectionsFor(ctx, string(databaseToCheck))
		if err != nil {
			log.Error(ctx, "Failed to connect to mongoDB to get the collections", err, logData)
			return errorWithMongoDBConnection
		}

		for _, collectionToCheck := range collectionsToCheck {
			if found := find(collectionsInDb, string(collectionToCheck)); !found {
				logData["Collection"] = string(collectionToCheck)
				log.Error(ctx, "Collection does not exist in the database", errorCollectionDoesNotExist, logData)
				return errorCollectionDoesNotExist
			}
		}
	}
	return nil
}

func find(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// Healthcheck calls service to check its health status
func (m *client) healthcheck(ctx context.Context) error {
	err := m.mongoConnection.Ping(ctx, timeOutInSeconds)
	if err != nil {
		log.Error(ctx, "Ping mongo", err)
		return err
	}

	return nil
}

// Checker calls an api health endpoint and  updates the provided CheckState accordingly
func (c *CheckMongoClient) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	err := c.healthcheck(ctx)
	if err != nil {
		state.Update(healthcheck.StatusCritical, err.Error(), 0)
		return nil
	}
	msg := healthyMessage

	if c.databaseCollection != nil && len(c.databaseCollection) > 0 {
		err = c.checkCollections(ctx)
		if err != nil {
			log.Error(ctx, "Error checking collections in mongo", err)
			state.Update(healthcheck.StatusCritical, err.Error(), 0)
			return nil
		}
		msg += healthyCollectionsMessage
	}

	state.Update(healthcheck.StatusOK, msg, 0)
	return nil
}
