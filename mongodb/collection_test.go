package mongodb_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	. "github.com/smartystreets/goconvey/convey"
)

type TestModel struct {
	ID              int                  `bson:"_id"`
	State           string               `bson:"state"`
	NewKey          int                  `bson:"new_key,omitempty"`
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *primitive.Timestamp `bson:"unique_timestamp,omitempty"`
}

type TestModelList []TestModel

func (tml TestModelList) AsInterfaceList() []interface{} {
	result := make([]interface{}, len(tml))

	for i, tm := range tml {
		result[i] = tm
	}

	return result
}

type Times struct {
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *primitive.Timestamp `bson:"unique_timestamp,omitempty"`
}

type testNamespacedModel struct {
	ID      int    `bson:"_id"`
	State   string `bson:"state"`
	NewKey  int    `bson:"new_key,omitempty"`
	Current Times  `bson:"current,omitempty"`
	Nixed   Times  `bson:"nixed,omitempty"`
}

func TestCollectionSuite(t *testing.T) {
	Convey("Given a connection to a real mongodb server with a test database and associated mongodb connection", t, func() {
		var (
			ctx          = context.Background()
			mongoVersion = "4.4.8"
			mongoServer  *mim.Server
			database     = "test-db"
			user         = "test-user"
			password     = "test-password"
			err          error
		)

		mongoServer, err = mim.Start(ctx, mongoVersion)
		So(err, ShouldBeNil)
		defer mongoServer.Stop(ctx)
		setupMongoConnectionTest(t, mongoServer, database, user, password)

		conn, err := mongoDriver.Open(getMongoDriverConfig(mongoServer, database, nil))
		So(err, ShouldBeNil)
		So(conn, ShouldNotBeNil)

		Convey("and a test collection", func() {
			collection := "test-collection"

			Convey("setup with test data for testing aggregate functionality", func() {
				testData := []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "first"}}

				if err := setUpTestData(ctx, conn, collection, testData); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}

				Convey("check data in original state", func() {
					res := []TestModel{}

					err := queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, testData)
				})

				Convey("Count gives the correct number of documents identified by the given filter", func() {

					n, err := conn.Collection(collection).Count(ctx, bson.M{"state": "first"})
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 2)

					n, err = conn.Collection(collection).Count(ctx, bson.M{"state": "second"})
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 1)
				})

				Convey("Distinct gives the correct list of distinct values for the given field", func() {

					l, err := conn.Collection(collection).Distinct(ctx, "_id", bson.M{})
					So(err, ShouldBeNil)
					So(l, ShouldResemble, []interface{}{int32(1), int32(2), int32(3)})

					l, err = conn.Collection(collection).Distinct(ctx, "state", bson.M{})
					So(err, ShouldBeNil)
					So(l, ShouldResemble, []interface{}{"first", "second"})

					n, err := conn.Collection(collection).Distinct(ctx, "state", bson.M{"state": "first"})
					So(err, ShouldBeNil)
					So(n, ShouldResemble, []interface{}{"first"})
				})

				Convey("Find returns 0 documents when no documents exist that satisfy the given filter", func() {
					res := []TestModel{}
					n, err := conn.Collection(collection).Find(context.Background(), bson.M{"state": "third"}, &res)
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 0)
					So(res, ShouldResemble, []TestModel{})
				})

				Convey("Find returns the expected documents that satisfy the given filter", func() {
					res := []TestModel{}

					n, err := conn.Collection(collection).Find(context.Background(), bson.M{"state": "first"}, &res)
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 2)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 3, State: "first"}})

					n, err = conn.Collection(collection).Find(context.Background(), bson.M{"state": "second"}, &res)
					So(err, ShouldBeNil)
					So(n, ShouldEqual, 1)
					So(res, ShouldResemble, []TestModel{{ID: 2, State: "second"}})
				})

				Convey("FindOne returns an ErrNoDocumentFound error when no documents exist which satisfy the given filter", func() {
					res := TestModel{}
					err := conn.Collection(collection).FindOne(context.Background(), bson.M{"state": "third"}, &res)
					So(err, ShouldResemble, mongoDriver.ErrNoDocumentFound)
				})

				Convey("FindOne returns the expected document that satisfies the given filter", func() {
					res := TestModel{}
					err := conn.Collection(collection).FindOne(context.Background(), bson.M{"state": "second"}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, TestModel{ID: 2, State: "second"})
				})

				Convey("FindOne returns only one document when multiple documents exist that satisfy the given filter", func() {
					res := TestModel{}
					err := conn.Collection(collection).FindOne(context.Background(), bson.M{"state": "first"}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldBeIn, []TestModel{{ID: 1, State: "first"}, {ID: 3, State: "first"}})
				})

				Convey("FindCursor returns an empty cursor when no documents exist that satisfy the given filter", func() {
					n, err := conn.Collection(collection).FindCursor(context.Background(), bson.M{"state": "third"})
					So(err, ShouldBeNil)
					So(n.Err(), ShouldBeNil)
					So(n.Next(ctx), ShouldBeFalse)
					So(n.Close(ctx), ShouldBeNil)
				})

				Convey("FindCursor returns a cursor with one document when only one document exists that satisfy the given filter", func() {
					res := TestModel{}

					n, err := conn.Collection(collection).FindCursor(context.Background(), bson.M{"state": "second"})
					So(err, ShouldBeNil)
					So(n.Err(), ShouldBeNil)
					So(n.Next(ctx), ShouldBeTrue)
					So(n.Decode(&res), ShouldBeNil)
					So(res, ShouldResemble, TestModel{ID: 2, State: "second"})
					So(n.Next(ctx), ShouldBeFalse)
					So(n.Close(ctx), ShouldBeNil)
				})

				Convey("FindCursor returns a cursor with multiple documents that satisfy the given filter", func() {
					res := [2]TestModel{}

					n, err := conn.Collection(collection).FindCursor(context.Background(), bson.M{"state": "first"})
					So(err, ShouldBeNil)
					So(n.Err(), ShouldBeNil)
					So(n.Next(ctx), ShouldBeTrue)
					So(n.Decode(&res[0]), ShouldBeNil)

					So(n.Next(ctx), ShouldBeTrue)
					So(n.Decode(&res[1]), ShouldBeNil)

					So(res, ShouldResemble, [2]TestModel{{ID: 1, State: "first"}, {ID: 3, State: "first"}})

					So(n.Next(ctx), ShouldBeFalse)
					So(n.Close(ctx), ShouldBeNil)
				})

			})

			Convey("setup with data for testing Insert functionality", func() {
				testData := []TestModel{{ID: 1, State: "first"}}

				if err := setUpTestData(ctx, conn, collection, testData); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}

				Convey("check data in original state", func() {
					res := []TestModel{}

					err := queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, testData)
				})

				Convey("Insert should return an error if inserting an object with a duplicate _id key", func() {

					ir, err := conn.Collection(collection).Insert(context.Background(), TestModel{ID: 1, NewKey: 123})
					So(err, ShouldHaveSameTypeAs, mongoDriver.Error{})
					So(mongo.IsDuplicateKeyError(err.(mongoDriver.Error).Unwrap()), ShouldBeTrue)
					So(ir, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}})
				})

				Convey("InsertOne should return an error if inserting an object with a duplicate _id key", func() {

					ir, err := conn.Collection(collection).InsertOne(context.Background(), TestModel{ID: 1, NewKey: 123})
					So(err, ShouldHaveSameTypeAs, mongoDriver.Error{})
					So(mongo.IsDuplicateKeyError(err.(mongoDriver.Error).Unwrap()), ShouldBeTrue)
					So(ir, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}})
				})

				Convey("Insert should insert the identified object as expected", func() {

					ir, err := conn.Collection(collection).Insert(context.Background(), TestModel{ID: 2, State: "second"})
					So(err, ShouldBeNil)
					So(ir.InsertedId, ShouldEqual, 2)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}})
				})

				Convey("InsertOne should insert the identified object as expected", func() {

					ir, err := conn.Collection(collection).InsertOne(context.Background(), TestModel{ID: 2, State: "second"})
					So(err, ShouldBeNil)
					So(ir.InsertedId, ShouldEqual, 2)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}})
				})

				Convey("InsertMany should return an error if inserting an object with a duplicate _id key", func() {

					ir, err := conn.Collection(collection).InsertMany(context.Background(), []interface{}{TestModel{ID: 1, NewKey: 123}, TestModel{ID: 2, State: "second"}})
					So(err, ShouldNotBeNil)
					So(err, ShouldHaveSameTypeAs, mongoDriver.Error{})
					So(mongo.IsDuplicateKeyError(err.(mongoDriver.Error).Unwrap()), ShouldBeTrue)
					So(ir, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}})
				})

				Convey("InsertMany should insert the given objects as expected", func() {

					ir, err := conn.Collection(collection).InsertMany(context.Background(), []interface{}{TestModel{ID: 2, State: "second"}, TestModel{ID: 3, State: "third"}})
					So(err, ShouldBeNil)
					So(ir.InsertedIds, ShouldResemble, []interface{}{int32(2), int32(3)})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "third"}})
				})
			})

			Convey("setup with data for testing Upsert functionality", func() {
				testData := []TestModel{{ID: 1, State: "first"}}

				if err := setUpTestData(ctx, conn, collection, testData); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}

				Convey("check data in original state", func() {
					res := []TestModel{}

					err := queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, testData)
				})

				Convey("UpsertById should insert an object if it doesn't exist", func() {

					ir, err := conn.Collection(collection).UpsertById(context.Background(), 2, bson.M{"$set": bson.M{"state": "second"}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 0, ModifiedCount: 0, UpsertedCount: 1, UpsertedID: int32(2)})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}})
				})

				Convey("Upsert should insert an object if it doesn't exist", func() {

					ir, err := conn.Collection(collection).Upsert(context.Background(), bson.M{"_id": 2}, bson.M{"$set": bson.M{"state": "second"}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 0, ModifiedCount: 0, UpsertedCount: 1, UpsertedID: int32(2)})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}})
				})

				Convey("UpsertOne should insert an object if it doesn't exist", func() {

					ir, err := conn.Collection(collection).UpsertOne(context.Background(), bson.M{"_id": 2}, bson.M{"$set": bson.M{"state": "second"}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 0, ModifiedCount: 0, UpsertedCount: 1, UpsertedID: int32(2)})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}})
				})

				Convey("UpsertById should update an object if it already exists", func() {

					ir, err := conn.Collection(collection).UpsertById(context.Background(), 1, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 1, ModifiedCount: 1, UpsertedCount: 0, UpsertedID: nil})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}})
				})

				Convey("Upsert should update an object if it already exists", func() {

					ir, err := conn.Collection(collection).Upsert(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 1, ModifiedCount: 1, UpsertedCount: 0, UpsertedID: nil})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}})
				})

				Convey("UpsertOne should update an object if it already exists", func() {

					ir, err := conn.Collection(collection).UpsertOne(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)
					So(ir, ShouldResemble, &mongoDriver.CollectionUpdateResult{MatchedCount: 1, ModifiedCount: 1, UpsertedCount: 0, UpsertedID: nil})

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}})
				})
			})

			Convey("setup with test data for testing Update functionality", func() {
				testData := []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "first"}}

				if err := setUpTestData(ctx, conn, collection, testData); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}

				Convey("check data in original state", func() {
					res := []TestModel{}

					err := queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, testData)
				})

				Convey("UpdateById should update the identified object as expected", func() {

					_, err := conn.Collection(collection).UpdateById(context.Background(), 1, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}, {ID: 2, State: "first"}})
				})

				Convey("Update should update the identified object as expected", func() {

					_, err := conn.Collection(collection).Update(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}, {ID: 2, State: "first"}})
				})

				Convey("UpdateOne should update the identified object as expected", func() {

					_, err := conn.Collection(collection).UpdateOne(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"new_key": 123}})
					So(err, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 123}, {ID: 2, State: "first"}})
				})

				Convey("UpdateOne using mongoDriver.WithUpdate to attach timestamps to update should give the expected results", func() {
					testStartTime := time.Now().Truncate(time.Second)
					res := TestModel{}

					update := bson.M{"$set": bson.M{"new_key": 321}}
					updateWithTimestamps, err := mongoDriver.WithUpdates(update)
					So(err, ShouldBeNil)
					So(updateWithTimestamps, ShouldResemble, bson.M{"$currentDate": bson.M{"last_updated": true, "unique_timestamp": bson.M{"$type": "timestamp"}}, "$set": bson.M{"new_key": 321}})

					_, err = conn.Collection(collection).UpdateOne(context.Background(), bson.M{"_id": 1}, updateWithTimestamps)
					So(err, ShouldBeNil)

					err = queryMongo(conn, collection, bson.M{"_id": 1}, &res)
					So(err, ShouldBeNil)
					So(res.ID, ShouldEqual, 1)
					So(res.State, ShouldEqual, "first")
					So(res.NewKey, ShouldEqual, 321)
					So(res.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
					// extract time part
					So(time.Unix(int64(res.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
				})

				Convey("UpdateOne with and update from mongoDriver.WithNamespacedUpdates() to attach namespaced timestamps to an update should give the expected results", func() {
					// ensure this testStartTime is greater than last
					time.Sleep(1010 * time.Millisecond)
					testStartTime := time.Now().Truncate(time.Second)
					res := testNamespacedModel{}

					update := bson.M{"$set": bson.M{"new_key": 1234}}
					updateWithTimestamps, err := mongoDriver.WithNamespacedUpdates(update, []string{"nixed.", "current."})
					So(err, ShouldBeNil)
					So(updateWithTimestamps, ShouldResemble, bson.M{
						"$currentDate": bson.M{
							"current.last_updated":     true,
							"current.unique_timestamp": bson.M{"$type": "timestamp"},
							"nixed.last_updated":       true,
							"nixed.unique_timestamp":   bson.M{"$type": "timestamp"},
						},
						"$set": bson.M{"new_key": 1234},
					})

					_, err = conn.Collection(collection).UpdateOne(context.Background(), bson.M{"_id": 1}, updateWithTimestamps)
					So(err, ShouldBeNil)

					err = queryMongo(conn, collection, bson.M{"_id": 1}, &res)
					So(err, ShouldBeNil)
					So(res.ID, ShouldEqual, 1)
					So(res.State, ShouldEqual, "first")
					So(res.NewKey, ShouldEqual, 1234)
					So(res.Current.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
					So(res.Nixed.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
					// extract time part

					So(time.Unix(int64(res.Current.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
					So(time.Unix(int64(res.Nixed.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
				})

				Convey("UpdateMany updates multiple matching documents", func() {
					_, err := conn.
						Collection(collection).
						UpdateMany(context.Background(), bson.M{"state": "first"}, bson.M{"$set": bson.M{"new_key": 9999}})
					So(err, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first", NewKey: 9999}, {ID: 2, State: "first", NewKey: 9999}})
				})

				Convey("UpdateMany updates no documents if none match", func() {
					_, err := conn.
						Collection(collection).
						UpdateMany(context.Background(), bson.M{"state": "second"}, bson.M{"$set": bson.M{"new_key": 9999}})
					So(err, ShouldBeNil)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "first"}})
				})
			})

			Convey("setup with data for testing Delete functionality", func() {
				testData := []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "second"}}

				if err := setUpTestData(ctx, conn, collection, testData); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}

				Convey("check data in original state", func() {
					res := []TestModel{}

					err := queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, testData)
				})

				Convey("Delete should delete the identified object as expected", func() {
					dr, err := conn.Collection(collection).Delete(context.Background(), bson.M{"state": "first"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 1)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("DeleteById should delete the identified object as expected", func() {
					dr, err := conn.Collection(collection).DeleteById(context.Background(), 1)
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 1)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("DeleteOne should delete the identified object as expected", func() {
					dr, err := conn.Collection(collection).DeleteOne(context.Background(), bson.M{"state": "first"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 1)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("Delete will chose one document to delete if the selector matches multiple documents", func() {
					dr, err := conn.Collection(collection).Delete(context.Background(), bson.M{"state": "second"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 1)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(len(res), ShouldEqual, 2)
					sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
					So(res[0], ShouldResemble, TestModel{ID: 1, State: "first"})
					So(res[1].ID, ShouldBeIn, []int{2, 3})
					So(res[1].State, ShouldEqual, "second")
				})

				Convey("DeleteOne will chose one document to delete if the selector matches multiple documents", func() {
					dr, err := conn.Collection(collection).DeleteOne(context.Background(), bson.M{"state": "second"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 1)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(len(res), ShouldEqual, 2)
					sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
					So(res[0], ShouldResemble, TestModel{ID: 1, State: "first"})
					So(res[1].ID, ShouldBeIn, []int{2, 3})
					So(res[1].State, ShouldEqual, "second")
				})

				Convey("Delete should return a deleted count of 0 with no error, if no document is found that matches the given selector", func() {
					dr, err := conn.Collection(collection).Delete(context.Background(), bson.M{"state": "third"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 0)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("DeleteById should return a deleted count of 0 if no document is found that matches the given selector", func() {
					dr, err := conn.Collection(collection).DeleteById(context.Background(), 4)
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 0)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("DeleteOne should return a deleted count of 0 if no document is found that matches the given selector", func() {
					dr, err := conn.Collection(collection).DeleteOne(context.Background(), bson.M{"state": "third"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 0)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})

				Convey("DeleteMany deletes the expected documents that match the given selector", func() {

					dr, err := conn.Collection(collection).DeleteMany(context.Background(), bson.M{"state": "second"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 2)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}})
				})

				Convey("DeleteMany returns a deleted count of 0 if no document is found that matches the given selector", func() {
					dr, err := conn.Collection(collection).DeleteMany(context.Background(), bson.M{"state": "third"})
					So(err, ShouldBeNil)
					So(dr.DeletedCount, ShouldEqual, 0)

					res := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &res)
					So(err, ShouldBeNil)
					So(res, ShouldResemble, []TestModel{{ID: 1, State: "first"}, {ID: 2, State: "second"}, {ID: 3, State: "second"}})
				})
			})
		})
	})
}

func queryMongo(mongoConnection *mongoDriver.MongoConnection, collection string, query bson.M, res interface{}) error {
	if err := mongoConnection.Collection(collection).FindOne(context.Background(), query, res); err != nil {
		return err
	}

	return nil
}

func queryCursor(mongoConnection *mongoDriver.MongoConnection, collection string, query bson.M, res interface{}) error {
	ctx := context.Background()
	cursor, err := mongoConnection.Collection(collection).FindCursor(context.Background(), query)
	if err != nil {
		return err
	}

	rawCursor, ok := cursor.(*mongo.Cursor)
	if !ok {
		return errors.New("not a mongo cursor")
	}

	return rawCursor.All(ctx, res)
}

func getMongoDriverConfig(mongoServer *mim.Server, database string, collections map[string]string) *mongoDriver.MongoDriverConfig {
	return &mongoDriver.MongoDriverConfig{
		ConnectTimeout:  5 * time.Second,
		QueryTimeout:    5 * time.Second,
		ClusterEndpoint: mongoServer.URI(),
		Database:        database,
		Collections:     collections,
	}
}

func setUpTestData(ctx context.Context, mongoConnection *mongoDriver.MongoConnection, collection string, docs TestModelList) error {
	if err := mongoConnection.DropDatabase(ctx); err != nil {
		return err
	}

	if _, err := mongoConnection.Collection(collection).InsertMany(ctx, docs.AsInterfaceList()); err != nil {
		return err
	}

	return nil
}
