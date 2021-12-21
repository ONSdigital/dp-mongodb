package mongodb

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

// Find set the find query
func (find *Find) Find(query interface{}) *Find {
	find.query = query
	return find
}

// Sort set the sort criteria
func (find *Find) Sort(sort interface{}) *Find {
	find.sort = sort
	return find
}

// Limit set the max number of records to retrieve
func (find *Find) Limit(limit int) *Find {
	find.limit = int64(limit)
	return find
}

// Skip set the number of records to skip when the query is run
func (find *Find) Skip(skip int) *Find {
	find.skip = int64(skip)
	return find
}

// Select specifies the fields to return
func (find *Find) Select(projection interface{}) *Find {
	find.projection = projection
	return find
}

// Count the number of records which match the find query
func (find *Find) Count(ctx context.Context) (int, error) {
	count := options.Count()

	if find.skip != 0 {
		count.SetSkip(find.skip)
	}

	if find.limit != 0 {
		count.SetLimit(find.limit)
	}

	docCount, err := find.collection.CountDocuments(ctx, find.query, count)
	return int(docCount), wrapMongoError(err)
}

// One finds a record which matches the find criteria
// Current FindOptions are limited to what is used: Sort, Skip, Projection
// Other exhaustive list of options are: AllowPartialResults,BatchSize,Collation,
// Comment,CursorType,Hint,Max, MaxAwaitTime, MaxTime ,Min,
// NoCursorTimeout,OplogReplay,ReturnKey,ShowRecordID,
// Snapshot,
// ref: https://github.com/mongodb/mongo-go-driver/blob/master/mongo/options/findoptions.go#L306
func (find *Find) One(ctx context.Context, val interface{}) error {
	findOneOptions := options.FindOne()

	if find.skip != 0 {
		findOneOptions.SetSkip(find.skip)
	}

	if find.sort != nil {
		findOneOptions.SetSort(find.sort)
	}

	if find.projection != nil {
		findOneOptions.SetProjection(find.projection)
	}

	result := find.collection.FindOne(ctx, find.query, findOneOptions)
	if result.Err() != nil {
		return wrapMongoError(result.Err())
	}

	return wrapMongoError(result.Decode(val))
}

// All return all the results for this query, you do not need to close the cursor after this call
func (find *Find) All(ctx context.Context, results interface{}) error {
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

	cursor, err := find.collection.Find(ctx, find.query, findOptions)
	if err != nil {
		return wrapMongoError(err)
	}

	return wrapMongoError(cursor.All(ctx, results))
}

// Distinct return only distinct records
func (find *Find) Distinct(ctx context.Context, fieldName string) ([]interface{}, error) {
	results, err := find.collection.Distinct(ctx, fieldName, find.query)

	return results, wrapMongoError(err)
}
