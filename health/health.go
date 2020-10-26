package health

//go:generate moq -out mock/health.go -pkg mock . Session DataLayer CollectionLayer

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/log"
	"github.com/globalsign/mgo"
)

// ServiceName mongodb
const ServiceName = "mongodb"

var (
	// HealthyMessage is the message that will be used in healthcheck when mongo is Healthy and all the collections exist
	HealthyMessage = "mongodb is OK and all expected collections exist"
)

//List of errors
var (
	ErrorCollectionDoesNotExist = errors.New("collection not found in database")
	ErrorWithMongoDBConnection  = errors.New("unable to connect with MongoDB")
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

// MongoSession is currently a mgo.Session
type MongoSession struct {
	*mgo.Session
}

// MongoDatabase wraps a mgo.Database to embed methods in models
type MongoDatabase struct {
	*mgo.Database
}

// MongoCollection wraps a mgo.Collection to embed methods in models
type MongoCollection struct {
	*mgo.Collection
}

// Check that the MongoSession satifies the the Session interface
var _ Session = (*MongoSession)(nil)

// Session is an interface that define the functions from mgo
type Session interface {
	Close()
	Copy() Session
	DB(name string) DataLayer
	Ping() error
}

// DataLayer is an interface that define the functions from mgo.db
type DataLayer interface {
	CollectionNames() ([]string, error)
	C(name string) CollectionLayer
}

// CollectionLayer is an interface that define the functions from mgo.Collection
type CollectionLayer interface {
	Find(query interface{}) *mgo.Query
	UpsertId(id interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	UpdateId(id interface{}, update interface{}) error
}

// Copy shadows *mgo.Copy to return a Session interface instead of *mgo.Session
func (s MongoSession) Copy() Session {
	return &MongoSession{Session: s.Session.Copy()}
}

// DB shadows *mgo.DB to return a DataLayer interface instead of *mgo.Database
func (s MongoSession) DB(name string) DataLayer {
	return &MongoDatabase{Database: s.Session.DB(name)}
}

// C shadows *mgo.Collection to return a CollectionLayer interface instead of *mgo.Collection
func (db MongoDatabase) C(name string) CollectionLayer {
	return &MongoCollection{Collection: db.Database.C(name)}
}

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo              Session
	serviceName        string
	databaseCollection map[Database][]Collection
}

// NewClient returns a new health check client using the given service
func NewClient(db Session) *Client {
	return NewClientWithCollections(db, nil)
}

// NewClientWithCollections returns a new health check client containing the collections using the given service
func NewClientWithCollections(db Session, clientDatabaseCollection map[Database][]Collection) *Client {
	return &Client{
		mongo:              db,
		serviceName:        ServiceName,
		databaseCollection: clientDatabaseCollection,
	}
}

func checkCollections(ctx context.Context, dbSession Session, databaseCollectionMap map[Database][]Collection) (err error) {

	for databaseToCheck, collectionsToCheck := range databaseCollectionMap {

		logData := log.Data{"Database": string(databaseToCheck)}
		collectionsInDb, err := dbSession.DB(string(databaseToCheck)).CollectionNames()
		if err != nil {
			log.Event(ctx, "Failed to connect to mongoDB to get the collections", log.ERROR, logData, log.Error(err))
			return ErrorWithMongoDBConnection
		}

		for _, collectionToCheck := range collectionsToCheck {
			if found := find(collectionsInDb, string(collectionToCheck)); !found {
				logData["Collection"] = string(collectionToCheck)
				log.Event(ctx, "Collection does not exist in the database", log.ERROR, logData, log.Error(ErrorCollectionDoesNotExist))
				return ErrorCollectionDoesNotExist
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
func (m *Client) Healthcheck(ctx context.Context) (res string, err error) {
	s := m.mongo.Copy()
	defer s.Close()
	res = m.serviceName
	err = s.Ping()
	if err != nil {
		log.Event(ctx, "Ping mongo", log.ERROR, log.Error(err))
		return
	}

	if m.databaseCollection != nil {
		err = checkCollections(ctx, s, m.databaseCollection)
		if err != nil {
			log.Event(ctx, "Error checking collections in mongo", log.ERROR, log.Error(err))
			return
		}
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
