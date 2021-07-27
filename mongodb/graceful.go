package mongodb

import (
	"context"

	"github.com/ONSdigital/log.go/v2/log"
	"go.mongodb.org/mongo-driver/mongo"
)

// Graceful represents an interface to the shutdown method
type Graceful interface {
	shutdown(ctx context.Context, mongoClient *mongo.Client, closedChannel chan bool)
}

type graceful struct{}

func (t graceful) shutdown(ctx context.Context, mongoClient *mongo.Client, closedChannel chan bool) {
	err := mongoClient.Disconnect(ctx)
	if err != nil {
		log.Warn(context.Background(), "Error in disconnecting from database", log.FormatErrors([]error{err}))
	}

	closedChannel <- true
	return
}
