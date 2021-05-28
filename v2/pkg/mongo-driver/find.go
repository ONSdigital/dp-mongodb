package mongo_driver

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Find struct {
	collection *mongo.Collection
	query      interface{}
	limit      int64
	skip       int64
	sort       interface{}
	projection interface{}
}

func newFind(collection *mongo.Collection, query interface{}) *Find {
	return &Find{collection, query, 0, 0, nil, nil}
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

func (find *Find) Select(projection interface{}) *Find {
	find.projection = projection
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

func (find *Find) One(ctx context.Context, val interface{}) error {
	result := find.collection.FindOne(ctx, find.query)

	if result.Err() != nil {
		return result.Err()
	}

	return result.Decode(val)
}

func (find *Find) Iter() *Cursor {
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

	if find.projection != nil {
		findOptions.SetProjection(find.projection)
	}

	return newCursor(find.collection, find.query, findOptions)
}

func (find *Find) IterAll(ctx context.Context, results interface{}) error {
	cursor := find.Iter()

	return wrapMongoError(cursor.All(ctx, results))
}

func (find *Find) Distinct(ctx context.Context, fieldName string) ([]interface{}, error) {
	return find.collection.Distinct(ctx, fieldName, find.query)
}
