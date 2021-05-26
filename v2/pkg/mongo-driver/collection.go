package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection struct {
	collection *mongo.Collection
}

type CollectionInsertResult struct {
	InsertedIds []interface{}
}

// MongoUpdateResult is the result type returned from UpdateOne, UpdateMany, and ReplaceOne operations.
// CollectionUpdateResult is the result type returned from UpdateOne, UpdateMany, and ReplaceOne operations.
type CollectionUpdateResult struct {
	MatchedCount  int64       // The number of documents matched by the filter.
	ModifiedCount int64       // The number of documents modified by the operation.
	UpsertedCount int64       // The number of documents upserted by the operation.
	UpsertedID    interface{} // The _id field of the upserted document, or nil if no upsert was done.
}

type CollectionDeleteResult struct {
	DeletedCount int64 // The number of records deleted
}

func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{collection}
}

func (c *Collection) Find(query interface{}) *Find {
	return newFind(c.collection, query)
}

func (c *Collection) Insert(ctx context.Context, documents []interface{}) (*CollectionInsertResult, error) {
	result, err := c.collection.InsertMany(ctx, documents)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	insertResult := &CollectionInsertResult{}
	copy(result.InsertedIDs, result.InsertedIDs)

	return insertResult, nil
}

func (c *Collection) Upsert(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	opts := options.Update().SetUpsert(true)

	updateResult, err := c.collection.UpdateByID(ctx, selector, update, opts)
	if err == nil {
		return &CollectionUpdateResult{
			MatchedCount:  updateResult.MatchedCount,
			ModifiedCount: updateResult.ModifiedCount,
			UpsertedCount: updateResult.UpsertedCount,
			UpsertedID:    updateResult.UpsertedID,
		}, nil
	}
	return nil, wrapMongoError(err)
}

func (c *Collection) UpsertId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.M{"_id": id}

	return c.Upsert(ctx, selector, update)
}

func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {

	err := c.collection.FindOne(ctx, filter).Decode(result)
	return wrapMongoError(err)
}

func (c *Collection) UpdateId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.M{"_id": id}

	return c.Update(ctx, selector, update)
}

func (c *Collection) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	updateResult, err := c.collection.UpdateByID(ctx, selector, update)

	if err == nil {
		return &CollectionUpdateResult{
			MatchedCount:  updateResult.MatchedCount,
			ModifiedCount: updateResult.ModifiedCount,
			UpsertedCount: updateResult.UpsertedCount,
			UpsertedID:    updateResult.UpsertedID,
		}, nil
	}
	return nil, wrapMongoError(err)
}

func (c *Collection) Remove(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {

	result, err := c.collection.DeleteMany(ctx, selector)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{result.DeletedCount}, nil
}

func (c *Collection) RemoveId(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	selector := bson.M{"_id": id}

	return c.Remove(ctx, selector)
}
