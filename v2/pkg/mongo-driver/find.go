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
func (find *Find) Limit(limit int64) *Find {
	find.limit = limit
	return find
}

// Skip set the number of records to skip when the query is run
func (find *Find) Skip(skip int64) *Find {
	find.skip = skip
	return find
}

// Select specifies the fields to return
func (find *Find) Select(projection interface{}) *Find {
	find.projection = projection
	return find
}

// Count the number of records which match the find query
func (find *Find) Count(ctx context.Context) (int64, error) {
	count := options.Count()

	if find.skip != 0 {
		count.SetSkip(find.skip)
	}

	if find.limit != 0 {
		count.SetLimit(find.limit)
	}

	docCount, err := find.collection.CountDocuments(ctx, find.query, count)
	return docCount, wrapMongoError(err)
}

// Find a record which matches the find criteria
func (find *Find) One(ctx context.Context, val interface{}) error {
	result := find.collection.FindOne(ctx, find.query)

	if result.Err() != nil {
		return wrapMongoError(result.Err())
	}

	result.Decode(val)

	return nil
}

// Iter return a cursor to iterate through the results
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

	return newCursor(newFindCursor(find.collection, find.query, findOptions))
}

// IterAll return all the results for this query, you do not need to close the cursor after
// this call
func (find *Find) IterAll(ctx context.Context, results interface{}) error {
	cursor := find.Iter()

	return wrapMongoError(cursor.All(ctx, results))
}

// Distinct return only distinct records
func (find *Find) Distinct(ctx context.Context, fieldName string) ([]interface{}, error) {
<<<<<<< HEAD
	results, err := find.collection.Distinct(ctx, fieldName, find.query)

	return results, wrapMongoError(err)
=======
	distinctData, err := find.collection.Distinct(ctx, fieldName, find.query)
	return distinctData, wrapMongoError(err)
>>>>>>> 5d2dc38707741418276233d2b47a81d8cbf70d67
}
