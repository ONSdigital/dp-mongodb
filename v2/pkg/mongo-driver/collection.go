package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection struct {
	collection *mongo.Collection
}

// MongoUpdateResult is the result type returned from UpdateOne, UpdateMany, and ReplaceOne operations.
type MongoUpdateResult struct {
	MatchedCount  int64       // The number of documents matched by the filter.
	ModifiedCount int64       // The number of documents modified by the operation.
	UpsertedCount int64       // The number of documents upserted by the operation.
	UpsertedID    interface{} // The _id field of the upserted document, or nil if no upsert was done.
}

func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{collection}
}

func (c *Collection) Find(query interface{}) *Find {
	return newFind(c.collection, query)
}

func (c *Collection) UpsertId(ctx context.Context, id interface{}, update interface{}) (*MongoUpdateResult, error) {
	opts := options.Update().SetUpsert(true)

	updateResult, err := c.collection.UpdateByID(ctx, id, update, opts)
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

func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {

	err := c.collection.FindOne(ctx, filter).Decode(result)
	return err
}

func (c *Collection) UpdateId(ctx context.Context, id interface{}, update interface{}) (*MongoUpdateResult, error) {
	updateResult, err := c.collection.UpdateByID(ctx, id, update)

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
