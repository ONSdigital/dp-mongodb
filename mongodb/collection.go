package mongodb

import (
	"context"

	lock "github.com/square/mongo-lock"
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

type Cursor interface {
	Close(ctx context.Context) error
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Err() error
}

// NewCollection creates a new collection
func NewCollection(collection *mongo.Collection) *Collection {
	return &Collection{collection}
}

// Must creates a new Must for the collection
func (c *Collection) Must() *Must {
	return newMust(c)
}

// Distinct returns the list of distinct values for the given field name in the collection
func (c *Collection) Distinct(ctx context.Context, fieldName string, filter interface{}) ([]interface{}, error) {

	results, err := c.collection.Distinct(ctx, fieldName, filter)

	return results, wrapMongoError(err)
}

// Count returns the number of documents in the collection that satisfy the given filter (which cannot be nil)
// Sort and Projection options are ignored. A Limit option <=0 is ignored, and a count of all documents is returned
func (c *Collection) Count(ctx context.Context, filter interface{}, opts ...FindOption) (int, error) {

	count, err := c.collection.CountDocuments(ctx, filter, newFindOptions(opts...).asDriverCountOption())

	return int(count), wrapMongoError(err)
}

// Find returns the total number of documents in the collection that satisfy the given filter (restricted by the
// given options), with the actual documents provided in the results parameter (which must be a non nil pointer
// to a slice of the expected document type)
// If no sort order option is provided a default sort order of 'ascending _id' is used (bson.M{"_id": 1})
func (c *Collection) Find(ctx context.Context, filter, results interface{}, opts ...FindOption) (int, error) {

	fo := newFindOptions(opts...)

	tc, err := c.collection.CountDocuments(ctx, filter)
	switch {
	case err != nil:
		return 0, err
	case tc == 0:
		return 0, nil
	case fo.limit < 0 || fo.skip >= tc,
		fo.limit == 0 && fo.obeyZeroLimit:
		return int(tc), nil
	}

	if fo.sort == nil {
		fo.sort = bson.M{"_id": 1}
	}
	cursor, err := c.collection.Find(ctx, filter, fo.asDriverFindOption())
	if err != nil {
		return 0, wrapMongoError(err)
	}

	return int(tc), wrapMongoError(cursor.All(ctx, results))
}

// FindCursor returns a mongo cursor iterating over the collection
// If no sort order option is provided a default sort order of 'ascending _id' is used (bson.M{"_id": 1})
func (c *Collection) FindCursor(ctx context.Context, filter interface{}, opts ...FindOption) (Cursor, error) {

	fo := newFindOptions(opts...)
	if fo.sort == nil {
		fo.sort = bson.M{"_id": 1}
	}
	cursor, err := c.collection.Find(ctx, filter, fo.asDriverFindOption())
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return cursor, nil
}

// FindOne locates a single document
func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}, opts ...FindOption) error {

	r := c.collection.FindOne(ctx, filter, newFindOptions(opts...).asDriverFindOneOption())
	if r.Err() != nil {
		return wrapMongoError(r.Err())
	}

	return wrapMongoError(r.Decode(result))
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

func (c *Collection) UpdateMany(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	updateResult, err := c.collection.UpdateMany(ctx, selector, update, options.Update())
	if err == nil {
		return &CollectionUpdateResult{
			MatchedCount:  int(updateResult.MatchedCount),
			ModifiedCount: int(updateResult.ModifiedCount),
		}, nil
	}

	return nil, wrapMongoError(err)
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
func (c *Collection) Aggregate(ctx context.Context, pipeline, results interface{}) error {
	cursor, err := c.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return wrapMongoError(err)
	}

	if err = cursor.All(ctx, results); err != nil {
		return wrapMongoError(err)
	}

	return nil
}

// NewLockClient creates a new Lock Client
func (c *Collection) NewLockClient() *lock.Client {
	return lock.NewClient(c.collection)
}
