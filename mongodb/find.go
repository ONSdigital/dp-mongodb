package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FindOption func(f *find)

var (
	Sort       = func(s interface{}) FindOption { return func(f *find) { f.sort = s } }
	Offset     = func(o int) FindOption { return func(f *find) { f.skip = int64(o) } }
	Limit      = func(l int) FindOption { return func(f *find) { f.limit = int64(l) } }
	Projection = func(p interface{}) FindOption { return func(f *find) { f.projection = p } }
)

type find struct {
	collection *mongo.Collection
	query      interface{}
	limit      int64
	skip       int64
	sort       interface{}
	projection interface{}
}

func newFind(collection *mongo.Collection, query interface{}, opts ...FindOption) *find {
	f := &find{collection: collection, query: query}
	for _, o := range opts {
		o(f)
	}

	return f
}

// Count the number of records which match the find query
func (find *find) count(ctx context.Context) (int, error) {
	var (
		docCount int64
		err      error
	)

	opts := options.Count().SetSkip(find.skip)
	if find.limit > 0 {
		opts.SetLimit(find.limit)
	}

	docCount, err = find.collection.CountDocuments(ctx, find.query, opts)

	return int(docCount), wrapMongoError(err)
}

func (find *find) one(ctx context.Context, val interface{}) error {

	result := find.collection.FindOne(ctx, find.query, options.FindOne().SetSort(find.sort).SetSkip(find.skip).SetProjection(find.projection))
	if result.Err() != nil {
		return wrapMongoError(result.Err())
	}

	return wrapMongoError(result.Decode(val))
}

// Iter return a cursor to iterate through the results
func (find *find) all(ctx context.Context, results interface{}) error {

	cursor, err := find.collection.Find(ctx, find.query, options.Find().SetSort(find.sort).SetSkip(find.skip).SetLimit(find.limit).SetProjection(find.projection))
	if err != nil {
		return wrapMongoError(err)
	}

	return wrapMongoError(cursor.All(ctx, results))
}
