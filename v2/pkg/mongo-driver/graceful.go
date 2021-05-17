package mongo_driver

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
)

// Graceful represents an interface to the shutdown method
type Graceful interface {
	shutdown(ctx context.Context, mongoClient *mongo.Client, closedChannel chan bool)
}

type graceful struct{}

func (t graceful) shutdown(ctx context.Context, mongoClient *mongo.Client, closedChannel chan bool) {
	err := mongoClient.Disconnect(ctx)
	if err != nil {
		// change to warn logs
		log.Fatalf("Error in disconnecting. Error: %s", err)
	}

	closedChannel <- true
	return
}
