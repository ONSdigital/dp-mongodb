package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ONSdigital/log.go/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	start    Graceful = graceful{}
	timeLeft          = 1000 * time.Millisecond
)

type MongoConnector interface {
	Ping(ctx context.Context) error
	C(collection string) *Collection
	Close(ctx context.Context) error
	GetCollectionsFor(ctx context.Context, database string) ([]string, error)
	GetConfiguredCollection() *Collection
	GetMongoCollection() *mongo.Collection
	DropDatabase(ctx context.Context) error
}

type MongoConnection struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongoConnection(client *mongo.Client, database string, collection string) *MongoConnection {
	m := &MongoConnection{client: client, database: database, collection: collection}
	return m
}

// Close represents mongo session closing within the context deadline
func (ms *MongoConnection) Close(ctx context.Context) error {
	closedChannel := make(chan bool)
	defer close(closedChannel)

	// Make a copy of timeLeft so that we don't modify the global var
	closeTimeLeft := timeLeft
	if deadline, ok := ctx.Deadline(); ok {
		// Add some time to timeLeft so case where ctx.Done in select
		// statement below gets called before time.After(timeLeft) gets called.
		// This is so the context error is returned over hardcoded error.
		closeTimeLeft = deadline.Sub(time.Now()) + (10 * time.Millisecond)
	}

	go func() {
		start.shutdown(ctx, ms.client, closedChannel)
		return
	}()

	select {
	case <-time.After(closeTimeLeft):
		return errors.New("closing mongo timed out")
	case <-closedChannel:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ms *MongoConnection) GetConfiguredCollection() *Collection {
	return NewCollection(ms.GetMongoCollection())
}

func (ms *MongoConnection) Ping(ctx context.Context, timeoutInSeconds time.Duration) error {
	connectionCtx, cancel := context.WithTimeout(ctx, timeoutInSeconds*time.Second)
	defer cancel()

	err := ms.client.Ping(connectionCtx, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to ping datastore: %v", err)
		log.Event(context.Background(), errMessage, log.ERROR, log.Error(err))
		return errors.New(errMessage)
	}
	return nil
}
func (ms *MongoConnection) ListCollectionsFor(ctx context.Context, database string) ([]string, error) {
	collectionNames, err := ms.
		client.
		Database(database).
		ListCollectionNames(ctx, bson.D{{"name",  bson.D{{"$ne", ""}}}})
	if err != nil {
		return nil, err
	}

	return collectionNames, nil
}

func (ms *MongoConnection) GetMongoCollection() *mongo.Collection {
	return ms.client.Database(ms.database).Collection(ms.collection)
}

func (ms *MongoConnection) C(collection string) *Collection {
	return NewCollection(ms.client.Database(ms.database).Collection(collection))
}

func (ms *MongoConnection) DropDatabase(ctx context.Context) error {
	return ms.client.Database(ms.database).Drop(ctx)
}
