package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ONSdigital/log.go/v2/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	start    Graceful = graceful{}
	timeLeft          = 1000 * time.Millisecond
)

type MongoConnector interface {
	Collection(collection string) *Collection
	ListCollectionsFor(ctx context.Context, database string) ([]string, error)
	DropDatabase(ctx context.Context) error
	RunCommand(ctx context.Context, runCommand interface{}) error
	Ping(ctx context.Context, timeoutInSeconds time.Duration) error
	Close(ctx context.Context) error
}

type MongoConnection struct {
	client   *mongo.Client
	database string
}

func NewMongoConnection(client *mongo.Client, database string) *MongoConnection {
	return &MongoConnection{client: client, database: database}
}

// Close represents mongo session closing within the context deadline
func (ms *MongoConnection) Close(ctx context.Context) error {
	closedChannel := make(chan bool)
	defer close(closedChannel)

	// Make a copy of timeLeft so that we don't modify the global var
	closeTimeLeft := timeLeft
	if deadline, ok := ctx.Deadline(); ok {
		// Add some time to timeLeft so case where ctx.Done in select
		// statement below gets called before time.NewTimer(closeTimeLeft) gets called.
		// This is so the context error is returned over hardcoded error.
		closeTimeLeft = deadline.Sub(time.Now()) + (10 * time.Millisecond)
	}

	go func() {
		start.shutdown(ctx, ms.client, closedChannel)
		return
	}()

	delay := time.NewTimer(closeTimeLeft)
	select {
	case <-delay.C:
		return errors.New("closing mongo timed out")
	case <-closedChannel:
		// Ensure timer is stopped and its resources are freed
		if !delay.Stop() {
			// if the timer has been stopped then read from the channel
			<-delay.C
		}
		return nil
	case <-ctx.Done():
		// Ensure timer is stopped and its resources are freed
		if !delay.Stop() {
			// if the timer has been stopped then read from the channel
			<-delay.C
		}
		return ctx.Err()
	}
}

func (ms *MongoConnection) Ping(ctx context.Context, timeoutInSeconds time.Duration) error {
	connectionCtx, cancel := context.WithTimeout(ctx, timeoutInSeconds*time.Second)
	defer cancel()

	err := ms.client.Ping(connectionCtx, nil)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to ping datastore: %v", err)
		log.Error(context.Background(), errMessage, err)
		return errors.New(errMessage)
	}
	return nil
}

func (ms *MongoConnection) ListCollectionsFor(ctx context.Context, database string) ([]string, error) {
	collectionNames, err := ms.
		client.
		Database(database).
		ListCollectionNames(ctx, bson.D{{Key: "name", Value: bson.D{{Key: "$ne", Value: ""}}}})
	if err != nil {
		return nil, err
	}

	return collectionNames, nil
}

func (ms *MongoConnection) d() *mongo.Database {
	return ms.client.Database(ms.database)
}

func (ms *MongoConnection) Collection(collection string) *Collection {
	return NewCollection(ms.d().Collection(collection))
}

func (ms *MongoConnection) DropDatabase(ctx context.Context) error {
	return ms.d().Drop(ctx)
}

// RunCommand executes the given command against the configured database.
// This is provided for tests only and no values are returned
func (ms *MongoConnection) RunCommand(ctx context.Context, runCommand interface{}) error {
	res := ms.d().RunCommand(ctx, runCommand)
	return res.Err()
}
