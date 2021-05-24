package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type Cursor struct {
	cursor *mongo.Cursor
}

func newCursor(cursor *mongo.Cursor) *Cursor {
	return &Cursor{cursor}
}

func (cursor *Cursor) Close(ctx context.Context) error {
	return cursor.cursor.Close(ctx)
}

func (cursor *Cursor) All(ctx context.Context, results interface{}) error {
	return wrapMongoError(cursor.cursor.All(ctx, results))
}

func (cursor *Cursor) Next(ctx context.Context) bool {
	return cursor.cursor.Next(ctx)
}

func (cursor *Cursor) TryNext(ctx context.Context) bool {
	return cursor.TryNext(ctx)
}
