package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Collection is a handle to a MongoDB collection
type Collection struct {
	collection *mongo.Collection
}

// CollectionInsertManyResult is the result type returned from InsertMany operations.
type CollectionInsertManyResult struct {
	InsertedIds []interface{} // inserted Ids
}

// CollectionUpdateResult is the result type returned from Update, UpdateById, Upsert and UpsertById operations.
type CollectionUpdateResult struct {
	MatchedCount  int         // The number of documents matched by the filter.
	ModifiedCount int         // The number of documents modified by the operation.
	UpsertedCount int         // The number of documents upserted by the operation.
	UpsertedID    interface{} // The _id field of the upserted document, or nil if no upsert was done.
}

// CollectionDeleteResult is the result type returned from Delete, DeleteById and DeleteMany operations.
type CollectionDeleteResult struct {
	DeletedCount int // The number of records deleted
}

// CollectionInsertResult is the result type return from Insert
type CollectionInsertResult struct {
	InsertedId interface{} // Id of the document inserted
}

// NewCollection creates a new collection
func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{collection}
}

// Must creates a new Must for the collection
func (c *Collection) Must() *Must {
	return newMust(c)
}

// Find returns a Find interface which can be used to either refine the criteria or retrieve a cursor
func (c *Collection) Find(query interface{}) *Find {
	return newFind(c.collection, query)
}

// Insert creates a single record
func (c *Collection) Insert(ctx context.Context, document interface{}) (*CollectionInsertResult, error) {
	result, err := c.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionInsertResult{result.InsertedID}, nil
}

// InsertMany adds a number of documents
func (c *Collection) InsertMany(ctx context.Context, documents []interface{}) (*CollectionInsertManyResult, error) {
	result, err := c.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	insertResult := &CollectionInsertManyResult{}
	insertResult.InsertedIds = result.InsertedIDs

	return insertResult, nil
}

// Upsert creates or updates a record located by a provided selector
func (c *Collection) Upsert(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	return c.updateRecord(ctx, selector, update, true)
}

// UpsertById creates or updates a record located by a provided Id selector
func (c *Collection) UpsertById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.D{{Key: "_id", Value: id}}
	return c.updateRecord(ctx, selector, update, true)
}

// UpdateById modifies a record located by a provided Id selector
func (c *Collection) UpdateById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	selector := bson.D{{Key: "_id", Value: id}}
	return c.updateRecord(ctx, selector, update, false)
}

// Update modifies a record located by a provided selector
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

// FindOne locates a single document
func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}) error {
	err := c.collection.FindOne(ctx, filter).Decode(result)
	return wrapMongoError(err)
}

// Delete deletes a record based on the provided selector
func (c *Collection) Delete(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := c.collection.DeleteOne(ctx, selector)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{int(result.DeletedCount)}, nil
}

// DeleteMany deletes records based on the provided selector
func (c *Collection) DeleteMany(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := c.collection.DeleteMany(ctx, selector)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{int(result.DeletedCount)}, nil
}

// DeleteById deletes a record based on the id selector
func (c *Collection) DeleteById(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	selector := bson.M{"_id": id}
	return c.Delete(ctx, selector)
}

// Aggregate starts a pipeline operation
func (c *Collection) Aggregate(pipeline interface{}) *Cursor {
	return newCursor(newAggregateCursor(c.collection, pipeline, options.Aggregate()))
}
