package mongo_driver

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var (
	start    Graceful = graceful{}
	timeLeft          = 1000 * time.Millisecond
)

type MongoConnector interface {
	Close(ctx context.Context) error
	UpsertId(ctx context.Context, id interface{}, update interface{}) (*MongoUpdateResult, error)
	UpdateId(ctx context.Context, id interface{}, update interface{}) error
	FindOne(ctx context.Context, filter interface{}, result interface{}) error
}

type MongoConnection struct {
	client     *mongo.Client
	database   string
	collection string
}

// MongoUpdateResult is the result type returned from UpdateOne, UpdateMany, and ReplaceOne operations.
type MongoUpdateResult struct {
	MatchedCount  int64       // The number of documents matched by the filter.
	ModifiedCount int64       // The number of documents modified by the operation.
	UpsertedCount int64       // The number of documents upserted by the operation.
	UpsertedID    interface{} // The _id field of the upserted document, or nil if no upsert was done.
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

func (ms *MongoConnection) getConfiguredCollection() *mongo.Collection {
	return ms.client.Database(ms.database).Collection(ms.collection)
}

func (ms *MongoConnection) UpsertId(ctx context.Context, id interface{}, update interface{}) (*MongoUpdateResult, error) {
	collection := ms.getConfiguredCollection()
	opts := options.Update().SetUpsert(true)

	updateResult, err := collection.UpdateByID(ctx, id, update, opts)
	if err == nil {
		return &MongoUpdateResult{
			MatchedCount:  updateResult.MatchedCount,
			ModifiedCount: updateResult.ModifiedCount,
			UpsertedCount: updateResult.UpsertedCount,
			UpsertedID:    updateResult.UpsertedID,
		}, nil
	}
	return nil, err
}

func (ms *MongoConnection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
	collection := ms.getConfiguredCollection()

	err := collection.FindOne(ctx, filter).Decode(result)
	return err
}
func (ms *MongoConnection) UpdateId(ctx context.Context, id interface{}, update interface{}) error {
	collection := ms.getConfiguredCollection()
	_, err := collection.UpdateByID(ctx, id, update)
	return err
}
