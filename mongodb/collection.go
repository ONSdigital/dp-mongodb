package mongodb

import (
	"context"

	lock "github.com/square/mongo-lock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
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
	DeletedCount int // The number of documents deleted
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

func getSpan(ctx context.Context, spanName string) trace.Span {
	tracer := otel.GetTracerProvider().Tracer("dp-mongodb")
	_, span := tracer.Start(ctx, spanName)
	return span
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
	span := getSpan(ctx, "collection.Distinct")
	defer span.End()

	results, err := c.collection.Distinct(ctx, fieldName, filter)

	return results, wrapMongoError(err)
}

// Count returns the number of documents in the collection that satisfy the given filter (which cannot be nil)
// Sort and Projection options are ignored. A Limit option <=0 is ignored, and a count of all documents is returned
func (c *Collection) Count(ctx context.Context, filter interface{}, opts ...FindOption) (int, error) {
	span := getSpan(ctx, "collection.Count")
	defer span.End()

	count, err := c.collection.CountDocuments(ctx, filter, newFindOptions(opts...).asDriverCountOption())

	return int(count), wrapMongoError(err)
}

// Find returns the total number of documents in the collection that satisfy the given filter (restricted by the
// given options), with the actual documents provided in the results parameter (which must be a non nil pointer
// to a slice of the expected document type)
// If no sort order option is provided a default sort order of 'ascending _id' is used (bson.M{"_id": 1})
func (c *Collection) Find(ctx context.Context, filter, results interface{}, opts ...FindOption) (int, error) {
	span := getSpan(ctx, "collection.Find")
	defer span.End()

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

// FindOne returns a single document in the collection that satisfies the given filter (restricted by the
// given options), with the actual document provided in the result parameter (which must be a non nil pointer
// to a document of the expected type)
// If no document could be found, an ErrNoDocumentFound error is returned
func (c *Collection) FindOne(ctx context.Context, filter interface{}, result interface{}, opts ...FindOption) error {
	span := getSpan(ctx, "collection.FindOne")
	defer span.End()

	r := c.collection.FindOne(ctx, filter, newFindOptions(opts...).asDriverFindOneOption())
	if r.Err() != nil {
		return wrapMongoError(r.Err())
	}

	return wrapMongoError(r.Decode(result))
}

// FindCursor returns a mongo cursor iterating over the collection
// If no sort order option is provided a default sort order of 'ascending _id' is used (bson.M{"_id": 1})
func (c *Collection) FindCursor(ctx context.Context, filter interface{}, opts ...FindOption) (Cursor, error) {
	span := getSpan(ctx, "collection.FindCursor")
	defer span.End()

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

// Insert creates a single document in the collection
// Deprecated: Use InsertOne
func (c *Collection) Insert(ctx context.Context, document interface{}) (*CollectionInsertResult, error) {
	span := getSpan(ctx, "collection.Insert")
	defer span.End()
	return c.InsertOne(ctx, document)
}

// InsertOne creates a single document in the collection
// The document must be the document to be inserted and cannot be nil.
// If the document does not have an _id field when transformed into BSON, one will be added automatically to the marshalled document.
// The _id can be retrieved from the InsertedId field of the returned CollectionInsertResult.
func (c *Collection) InsertOne(ctx context.Context, document interface{}) (*CollectionInsertResult, error) {
	span := getSpan(ctx, "InsertOne")
	defer span.End()
	result, err := c.collection.InsertOne(ctx, document)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionInsertResult{result.InsertedID}, nil
}

// InsertMany creates multiple documents in the collection
// The documents must be a slice of documents to insert and slice cannot be nil or empty. The elements must all be non-nil.
// For any document that does not have an _id field when transformed into BSON, one will be added automatically to the marshalled document.
// The _id values for the inserted documents can be retrieved from the InsertedIds field of the returned CollectionInsertManyResult.
func (c *Collection) InsertMany(ctx context.Context, documents []interface{}) (*CollectionInsertManyResult, error) {
	span := getSpan(ctx, "collection.InsertMany")
	defer span.End()
	result, err := c.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	insertResult := &CollectionInsertManyResult{}
	insertResult.InsertedIds = result.InsertedIDs

	return insertResult, nil
}

// Upsert creates or updates a document located by the provided selector
// Deprecated: Use UpsertOne
func (c *Collection) Upsert(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.Upsert")
	defer span.End()
	return c.UpsertOne(ctx, selector, update)
}

// UpsertById creates or updates a document located by the provided id selector
// Deprecated: Use UpsertOne
func (c *Collection) UpsertById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.UpsertById")
	defer span.End()
	return c.UpsertOne(ctx, bson.M{"_id": id}, update)
}

// UpsertOne creates or updates a document located by the provided selector
// The selector must be a document containing query operators and cannot be nil.
// The update must be a document containing update operators and cannot be nil or empty.
// If the selector does not match any documents, the update document is inserted into the collection.
// If the selector matches multiple documents, one will be selected from the matched set, updated and a CollectionUpdateResult with a MatchedCount of 1 will be returned.
func (c *Collection) UpsertOne(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.UpsertOne")
	defer span.End()
	return c.updateRecord(ctx, selector, update, true)
}

// UpdateById modifies a single document located by the provided id selector
// Deprecated: Use UpdateOne
func (c *Collection) UpdateById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.UpdateById")
	defer span.End()
	return c.UpdateOne(ctx, bson.M{"_id": id}, update)
}

// Update modifies a single document located by the provided selector
// Deprecated: Use UpdateOne
func (c *Collection) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.Update")
	defer span.End()
	return c.UpdateOne(ctx, selector, update)
}

// UpdateOne modifies a single document located by the provided selector
// The selector must be a document containing query operators and cannot be nil.
// The update must be a document containing update operators and cannot be nil or empty.
// If the selector does not match any documents, the operation will succeed and a CollectionUpdateResult with a MatchedCount of 0 will be returned.
// If the selector matches multiple documents, one will be selected from the matched set, updated and a CollectionUpdateResult with a MatchedCount of 1 will be returned.
func (c *Collection) UpdateOne(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "collection.UpdateOne")
	defer span.End()
	return c.updateRecord(ctx, selector, update, false)
}

// UpdateMany modifies multiple documents located by the provided selector
// The selector must be a document containing query operators and cannot be nil.
// The update must be a document containing update operators and cannot be nil or empty.
// If the selector does not match any documents, the operation will succeed and a CollectionUpdateResult with a MatchedCount of 0 will be returned.
func (c *Collection) UpdateMany(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	span := getSpan(ctx, "UpdateMany")
	defer span.End()
	updateResult, err := c.collection.UpdateMany(ctx, selector, update, options.Update())
	if err == nil {
		return &CollectionUpdateResult{
			MatchedCount:  int(updateResult.MatchedCount),
			ModifiedCount: int(updateResult.ModifiedCount),
		}, nil
	}

	return nil, wrapMongoError(err)
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

// Delete deletes a single document based on the provided selector
// Deprecated: Use DeleteOne instead
func (c *Collection) Delete(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	span := getSpan(ctx, "collection.Delete")
	defer span.End()
	return c.DeleteOne(ctx, selector)
}

// DeleteById deletes a document based on the provided id selector
// Deprecated: Use DeleteOne
func (c *Collection) DeleteById(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	span := getSpan(ctx, "collection.DeleteById")
	defer span.End()
	return c.DeleteOne(ctx, bson.M{"_id": id})
}

// DeleteOne deletes a single document based on the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// If the selector does not match any documents, the operation will succeed and a CollectionDeleteResult with a DeletedCount of 0 will be returned.
// If the selector matches multiple documents, one will be selected from the matched set, deleted and a CollectionDeleteResult with a DeletedCount of 1 will be returned.
func (c *Collection) DeleteOne(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	span := getSpan(ctx, "collection.DeleteOne")
	defer span.End()
	result, err := c.collection.DeleteOne(ctx, selector)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{int(result.DeletedCount)}, nil
}

// DeleteMany deletes multiple documents based on the provided selector
// The selector must be a document containing query operators and cannot be nil.
// If the selector does not match any documents, the operation will succeed and a CollectionDeleteResult with a DeletedCount of 0 will be returned.
func (c *Collection) DeleteMany(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	span := getSpan(ctx, "collection.DeleteMany")
	defer span.End()
	result, err := c.collection.DeleteMany(ctx, selector)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	return &CollectionDeleteResult{int(result.DeletedCount)}, nil
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
