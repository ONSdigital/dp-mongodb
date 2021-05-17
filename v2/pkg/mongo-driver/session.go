package mongo_driver

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

var (
	start    Graceful = graceful{}
	timeLeft          = 1000 * time.Millisecond
)

type MongoSessioner interface {
	Close(ctx context.Context) error
}

type MongoSession struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewMongoSession(client *mongo.Client, database string, collection string) *MongoSession {
	return &MongoSession{client: client, database: database, collection: collection}
}

// Close represents mongo session closing within the context deadline
func (mc *MongoSession) Close(ctx context.Context) error {
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
		start.shutdown(ctx, mc.client, closedChannel)
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
