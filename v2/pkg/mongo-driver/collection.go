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

// CollectionInsertResult is the result type returned from Insert
type CollectionInsertResult struct {
	InsertedIds []interface{} // inserted Ids
}

// CollectionUpdateResult is the result type returned from UpdateOne, UpdateMany, and ReplaceOne operations.
type CollectionUpdateResult struct {
	MatchedCount  int         // The number of documents matched by the filter.
	ModifiedCount int         // The number of documents modified by the operation.
	UpsertedCount int         // The number of documents upserted by the operation.
	UpsertedID    interface{} // The _id field of the upserted document, or nil if no upsert was done.
}

type CollectionDeleteResult struct {
	DeletedCount int // The number of records deleted
}

// CollectionInsertOneResult is the result type return from InsertOne
type CollectionInsertOneResult struct {
	InsertedId interface{} // Id of the document inserted
}

func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{collection}
}

func (c *Collection) Must() *Must {
	return newMust(c)
}

// Find returns a Find interface which can be used to either refine the criteria or retrieve a cursor
func (c *Collection) Find(query interface{}) *Find {
	return newFind(c.collection, query)
}

// Insert adds a number of documents
func (c *Collection) Insert(ctx context.Context, documents []interface{}) (*CollectionInsertResult, error) {
	result, err := c.collection.InsertMany(ctx, documents)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	insertResult := &CollectionInsertResult{}

	insertResult.InsertedIds = result.InsertedIDs

	return insertResult, nil
}

// Upsert creates or updates records located by a provided selector
func (c *Collection) Upsert(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	return c.updateRecord(ctx, selector, update, true)
}

// UpsertId creates or updates records located by a provided Id selector
func (c *Collection) UpsertId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.D{{"_id", id}}

	return c.updateRecord(ctx, selector, update, true)
}

// UpdateId modifies records located by a provided Id selector
func (c *Collection) UpdateId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.D{{"_id", id}}

	return c.updateRecord(ctx, selector, update, false)
}

// Update modifies records located by a provided selector
func (c *Collection) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	return c.updateRecord(ctx, selector, update, false)
}

func (c *Collection) updateRecord(ctx context.Context, selector interface{}, update interface{}, upsert bool) (*CollectionUpdateResult, error) {
	opts := options.Update()

	if upsert {
		opts.SetUpsert(true)
	}

	updateResult, err := c.collection.UpdateOne(ctx, selector, update, opts)

	if err == nil {
		return &CollectionUpdateResult{
			MatchedCount:  int(updateResult.MatchedCount),
			ModifiedCount: int(updateResult.ModifiedCount),
			UpsertedCount: int(updateResult.UpsertedCount),
			UpsertedID:    updateResult.UpsertedID,
		}, nil
	}
	return nil, wrapMongoError(err)
}

// InsertOne creates a single record
func (c *Collection) InsertOne(ctx context.Context, document interface{}) (*CollectionInsertOneResult, error) {
	result, err := c.collection.InsertOne(ctx, document)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionInsertOneResult{result.InsertedID}, nil
}

// FindOne locates a single document
func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {

	err := c.collection.FindOne(ctx, filter).Decode(result)
	return wrapMongoError(err)
}

// Remove deletes records based on the provided selector
func (c *Collection) Remove(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {

	result, err := c.collection.DeleteMany(ctx, selector)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{int(result.DeletedCount)}, nil
}

// RemoveId deletes record based on the id selector
func (c *Collection) RemoveId(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	selector := bson.M{"_id": id}

	return c.Remove(ctx, selector)
}

// Aggreate start a pipeline operation
func (c *Collection) Aggregate(pipeline interface{}) *Cursor {
	return newCursor(newAggregateCursor(c.collection, pipeline, options.Aggregate()))
}
