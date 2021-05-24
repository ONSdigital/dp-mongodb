package mongo_driver

import (
	"context"
	"github.com/ONSdigital/log.go/log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Find struct {
	collection *mongo.Collection
	query      interface{}
	limit      int64
	skip       int64
	sort       interface{}
}

func newFind(collection *mongo.Collection, query interface{}) *Find {
	return &Find{collection, query, 0, 0, nil}
}

func (find *Find) Find(query interface{}) *Find {
	find.query = query
	return find
}

func (find *Find) Sort(sort interface{}) *Find {
	find.sort = sort
	return find
}

func (find *Find) Limit(limit int64) *Find {
	find.limit = limit
	return find
}

func (find *Find) Skip(skip int64) *Find {
	find.skip = skip
	return find
}

func (find *Find) Count(ctx context.Context) (int64, error) {
	count := options.Count()

	if find.skip != 0 {
		count.SetSkip(find.skip)
	}

	if find.limit != 0 {
		count.SetLimit(find.limit)
	}

	return find.collection.CountDocuments(ctx, find.query, count)
}

func (find *Find) Iter(ctx context.Context) (*Cursor, error) {
	findOptions := options.Find()

	if find.skip != 0 {
		findOptions.SetSkip(find.skip)
	}

	if find.limit != 0 {
		findOptions.SetLimit(find.limit)
	}

	if find.sort != nil {
		findOptions.SetSort(find.sort)
	}

	cursor, err := find.collection.Find(ctx, find.query, findOptions)

	if err != nil {
		return nil, wrapMongoError(err)
	}

	return newCursor(cursor), nil
}

func (find *Find) IterAll(ctx context.Context, results interface{}) error {
	cursor, err := find.Iter(ctx)
	if err != nil {
		return wrapMongoError(err)
	}

	return cursor.All(ctx, results)
}
