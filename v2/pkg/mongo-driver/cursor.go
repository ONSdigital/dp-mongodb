package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateCursor interface to create a unique cursor
type CreateCursor interface {
	create(context.Context) (*mongo.Cursor, error)
}

// CreateFindCursor creates a find cursor
type CreateFindCursor struct {
	collection  *mongo.Collection
	query       interface{}
	findOptions *options.FindOptions
}

func newFindCursor(collection *mongo.Collection, query interface{}, findOptions *options.FindOptions) CreateCursor {
	return &CreateFindCursor{collection, query, findOptions}
}

func (createFindCursor *CreateFindCursor) create(ctx context.Context) (*mongo.Cursor, error) {
	return createFindCursor.collection.Find(ctx, createFindCursor.query, createFindCursor.findOptions)
}

// CreateAggreateCursor creates a cursor used with aggregation
type CreateAggregateCursor struct {
	collectiom *mongo.Collection
	pipeline   interface{}
	options    *options.AggregateOptions
}

func newAggregateCursor(collection *mongo.Collection, pipeline interface{}, options *options.AggregateOptions) CreateCursor {
	return &CreateAggregateCursor{collection, pipeline, options}
}

func (createAggregateCursor *CreateAggregateCursor) create(ctx context.Context) (*mongo.Cursor, error) {
	return createAggregateCursor.collectiom.Aggregate(ctx, createAggregateCursor.pipeline, createAggregateCursor.options)
}

type Cursor struct {
	createCursor CreateCursor
	cursor       *mongo.Cursor
	lastError    error
}

func newCursor(createCursor CreateCursor) *Cursor {
	return &Cursor{createCursor, nil, nil}
}

// Close closes a cursor
func (cursor *Cursor) Close(ctx context.Context) error {
	return cursor.cursor.Close(ctx)
}

// All returns all the results for this cursor
func (cursor *Cursor) All(ctx context.Context, results interface{}) error {
	mongoCursor, err := cursor.createCursor.create(ctx)

	if err != nil {
		cursor.lastError = err
		return wrapMongoError(err)
	}

	err = mongoCursor.All(ctx, results)

	return wrapMongoError(err)
}

// Next returns the next available record which must be available
func (cursor *Cursor) Next(ctx context.Context) bool {
	if cursor.cursor == nil {
		var err error
		cursor.cursor, err = cursor.createCursor.create(ctx)

		if err != nil {
			cursor.lastError = err
			return false
		}
	}

	return cursor.cursor.Next(ctx)
}

// TryNext trys to retrieve the next available record
func (cursor *Cursor) TryNext(ctx context.Context) bool {
	if cursor.cursor == nil {
		var err error
		cursor.cursor, err = cursor.createCursor.create(ctx)

		if err != nil {
			cursor.lastError = err
			return false
		}
	}

	return cursor.TryNext(ctx)
}

// Err returns the last error which occured for this cursor
func (cursor *Cursor) Err() error {
	if cursor.cursor == nil {
		return nil
	}
	if cursor.lastError != nil {
		return wrapMongoError(cursor.lastError)
	}

	return wrapMongoError(cursor.cursor.Err())
}
