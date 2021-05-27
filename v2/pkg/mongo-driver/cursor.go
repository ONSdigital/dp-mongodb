package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Cursor struct {
	collection  *mongo.Collection
	query       interface{}
	findOptions *options.FindOptions
	cursor      *mongo.Cursor
	lastError   error
}

func newCursor(collection *mongo.Collection, query interface{}, findOptions *options.FindOptions) *Cursor {
	return &Cursor{collection, query, findOptions, nil, nil}
}

func (cursor *Cursor) Close(ctx context.Context) error {
	return cursor.cursor.Close(ctx)
}

func (cursor *Cursor) All(ctx context.Context, results interface{}) error {
	findCursor, err := cursor.collection.Find(ctx, cursor.query, cursor.findOptions)

	if err != nil {
		cursor.lastError = err
		return wrapMongoError(err)
	}

	err = findCursor.All(ctx, results)

	return wrapMongoError(err)
}

func (cursor *Cursor) Next(ctx context.Context) bool {
	if cursor.cursor == nil {
		var err error
		cursor.cursor, err = cursor.collection.Find(ctx, cursor.query, cursor.findOptions)

		if err != nil {
			cursor.lastError = err
			return false
		}
	}

	return cursor.cursor.Next(ctx)
}

func (cursor *Cursor) TryNext(ctx context.Context) bool {
	if cursor.cursor == nil {
		var err error
		cursor.cursor, err = cursor.collection.Find(ctx, cursor.query, cursor.findOptions)

		if err != nil {
			cursor.lastError = err
			return false
		}
	}

	return cursor.TryNext(ctx)
}

func (cursor *Cursor) Err() error {
	if cursor.cursor == nil {
		return nil
	}
	if cursor.lastError != nil {
		return wrapMongoError(cursor.lastError)
	}

	return wrapMongoError(cursor.cursor.Err())
}
