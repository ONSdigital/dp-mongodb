package mongodb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TestModel struct {
	State           string               `bson:"state"`
	NewKey          int                  `bson:"new_key,omitempty"`
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *primitive.Timestamp `bson:"unique_timestamp,omitempty"`
}

type Times struct {
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *primitive.Timestamp `bson:"unique_timestamp,omitempty"`
}

type testNamespacedModel struct {
	State   string `bson:"state"`
	NewKey  int    `bson:"new_key,omitempty"`
	Currant Times  `bson:"currant,omitempty"`
	Nixed   Times  `bson:"nixed,omitempty"`
}

func TestSuite(t *testing.T) {
	Convey("Given a connection to a mongodb server set up with a database and a set of collections", t, func() {
		ctx := context.Background()
		database := "testDB"
		collection := "testCollection"
		collections := map[string]string{collection: "test-collection"}
		mongoServer, err := mim.Start(ctx, "4.4.8")
		if err != nil {
			t.Fatalf("failed to start mongo server: %v", err)
		}
		defer mongoServer.Stop(ctx)

		driverConfig := getMongoDriverConfig(mongoServer, database, collections)
		conn, err := mongoDriver.Open(driverConfig)
		So(err, ShouldBeNil)
		So(conn, ShouldNotBeNil)

		Convey("With some test data", func() {
			if err := setUpTestData(conn, collection); err != nil {
				t.Fatalf("failed to insert test data, skipping tests: %v", err)
			}

			Convey("check data in original state", func() {
				res := TestModel{}

				err := queryMongo(conn, collection, bson.M{"_id": 1}, &res)
				So(err, ShouldBeNil)
				So(res.State, ShouldEqual, "first")
			})

			Convey("check data after plain Update", func() {
				res := TestModel{}
				_, err := conn.Collection(collection).UpdateById(context.Background(), 1, bson.M{"$set": bson.M{"new_key": 123}})
				So(err, ShouldBeNil)

				err = queryMongo(conn, collection, bson.M{"_id": 1}, &res)
				So(err, ShouldBeNil)
				So(res.State, ShouldEqual, "first")
				So(res.NewKey, ShouldEqual, 123)
			})

			Convey("check data with Update with new dates", func() {
				testStartTime := time.Now().Truncate(time.Second)
				res := TestModel{}

				update := bson.M{"$set": bson.M{"new_key": 321}}
				updateWithTimestamps, err := mongoDriver.WithUpdates(update)
				So(err, ShouldBeNil)
				So(updateWithTimestamps, ShouldResemble, bson.M{"$currentDate": bson.M{"last_updated": true, "unique_timestamp": bson.M{"$type": "timestamp"}}, "$set": bson.M{"new_key": 321}})

				_, err = conn.Collection(collection).UpdateById(context.Background(), 1, updateWithTimestamps)
				So(err, ShouldBeNil)

				err = queryMongo(conn, collection, bson.M{"_id": 1}, &res)
				So(err, ShouldBeNil)
				So(res.State, ShouldEqual, "first")
				So(res.NewKey, ShouldEqual, 321)
				So(res.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
				// extract time part
				So(time.Unix(int64(res.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
			})

			Convey("check data with Update with new Namespaced dates", func() {
				// ensure this testStartTime is greater than last
				time.Sleep(1010 * time.Millisecond)
				testStartTime := time.Now().Truncate(time.Second)
				res := testNamespacedModel{}

				update := bson.M{"$set": bson.M{"new_key": 1234}}
				updateWithTimestamps, err := mongoDriver.WithNamespacedUpdates(update, []string{"nixed.", "currant."})
				So(err, ShouldBeNil)
				So(updateWithTimestamps, ShouldResemble, bson.M{
					"$currentDate": bson.M{
						"currant.last_updated":     true,
						"currant.unique_timestamp": bson.M{"$type": "timestamp"},
						"nixed.last_updated":       true,
						"nixed.unique_timestamp":   bson.M{"$type": "timestamp"},
					},
					"$set": bson.M{"new_key": 1234},
				})

				_, err = conn.Collection(collection).UpdateById(context.Background(), 1, updateWithTimestamps)
				So(err, ShouldBeNil)

				err = queryMongo(conn, collection, bson.M{"_id": 1}, &res)
				So(err, ShouldBeNil)
				So(res.State, ShouldEqual, "first")
				So(res.NewKey, ShouldEqual, 1234)
				So(res.Currant.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
				So(res.Nixed.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
				// extract time part

				So(time.Unix(int64(res.Currant.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
				So(time.Unix(int64(res.Nixed.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
			})

			Convey("UpsertId should insert if not exists", func() {
				_, err := conn.
					Collection(collection).
					UpsertById(context.Background(), 4, bson.M{"$set": bson.M{"new_key": 456}})
				So(err, ShouldBeNil)

				res := TestModel{}

				err = queryMongo(conn, collection, bson.M{"_id": 4}, &res)
				So(err, ShouldBeNil)
				So(res.NewKey, ShouldEqual, 456)
			})

			Convey("UpsertId should update if  exists", func() {
				_, err := conn.
					Collection(collection).
					UpsertById(context.Background(), 3, bson.M{"$set": bson.M{"new_key": 789}})
				So(err, ShouldBeNil)

				res := TestModel{}
				err = queryMongo(conn, collection, bson.M{"_id": 3}, &res)
				So(err, ShouldBeNil)
				So(res.NewKey, ShouldEqual, 789)
			})

			Convey("UpdateId should update data if document exists", func() {
				_, err := conn.Collection(collection).UpdateById(context.Background(), 3, bson.M{"$set": bson.M{"new_key": 7892}})
				So(err, ShouldBeNil)

				res := TestModel{}

				err = queryMongo(conn, collection, bson.M{"_id": 3}, &res)
				So(err, ShouldBeNil)
				So(res.NewKey, ShouldEqual, 7892)
			})

			Convey("FindOne should find data if document exists", func() {
				res := TestModel{}
				err := conn.
					Collection(collection).
					FindOne(context.Background(), bson.M{"_id": 3}, &res)
				So(err, ShouldBeNil)

				So(res.State, ShouldEqual, "third")
			})
		})

		Convey("When we run a valid command", func() {
			err = conn.RunCommand(context.Background(), bson.D{
				{Key: "createUser", Value: "test-user"},
				{Key: "pwd", Value: "password"},
				{Key: "roles", Value: []bson.M{}}})
			Convey("Then there are no errors", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When we run an invalid command", func() {
			err = conn.RunCommand(context.Background(), bson.D{{Key: "createUser", Value: "test-user"}})
			Convey("Then there is an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func getMongoDriverConfig(mongoServer *mim.Server, database string, collections map[string]string) *mongoDriver.MongoDriverConfig {
	return &mongoDriver.MongoDriverConfig{
		ConnectTimeout:  5 * time.Second,
		QueryTimeout:    5 * time.Second,
		ClusterEndpoint: fmt.Sprintf("localhost:%d", mongoServer.Port()),
		Database:        database,
		Collections:     collections,
	}
}

func setUpTestData(mongoConnection *mongoDriver.MongoConnection, collection string) error {
	ctx := context.Background()
	for i, data := range getTestData() {
		if _, err := mongoConnection.
			Collection(collection).
			UpsertById(ctx, i+1, bson.M{"$set": data}); err != nil {
			return err
		}
	}
	return nil
}

func getTestData() []bson.M {
	return []bson.M{
		{
			"state": "first",
		},
		{
			"state": "second",
		},
		{
			"state": "third",
		},
	}
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
