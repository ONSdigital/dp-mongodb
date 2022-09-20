package mongodb_test

import (
	"context"
	"testing"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
)

func TestCollection(t *testing.T) {
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

		Convey("UpdateMany", func() {
			Convey("When given invalid selector and update", func() {
				Convey("Then an error is returned", func() {
					_, err := conn.Collection(collection).UpdateMany(context.Background(), "wrong", "broken")
					So(err, ShouldNotBeNil)
				})
			})
		})
		Convey("FindCursor", func() {
			Convey("With some test data", func() {
				if err := setUpTestData(conn, collection); err != nil {
					t.Fatalf("failed to insert test data, skipping tests: %v", err)
				}
				Convey("All data are returned on no query", func() {
					result := []TestModel{}
					err = queryCursor(conn, collection, bson.M{}, &result)
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 3)
				})
				Convey("Data are filtered on query", func() {
					result := []TestModel{}
					err = queryCursor(conn, collection, bson.M{"state": "first"}, &result)
					So(err, ShouldBeNil)
					So(len(result), ShouldEqual, 1)
				})
			})
		})
	})
}
