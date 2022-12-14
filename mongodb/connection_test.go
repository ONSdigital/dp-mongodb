package mongodb_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConnectionSuite(t *testing.T) {
	Convey("Given a connection to a real mongodb server", t, func() {
		var (
			ctx          = context.Background()
			mongoVersion = "4.4.8"
			mongoServer  *mim.Server
			mongoClient  *mongo.Client
			database     = "test-db"
			user         = "test-user"
			password     = "test-password"
			err          error
		)

		mongoServer, err = mim.Start(ctx, mongoVersion)
		if err != nil {
			t.Fatalf("failed to start mongo server: %v", err)
		}
		defer mongoServer.Stop(ctx)
		mongoClient = setupMongoConnectionTest(t, mongoServer, database, user, password)

		Convey("with an associated mongodb connection", func() {
			conn, err := mongoDriver.Open(getMongoDriverConfig(mongoServer, database, nil))
			So(err, ShouldBeNil)
			So(conn, ShouldNotBeNil)

			Convey("When we call Ping", func() {
				So(conn.Ping(ctx, 10*time.Millisecond), ShouldBeNil)
			})

			Convey("For Connection", func() {

				Convey("When called it returns a handle to a Collection", func() {
					collectionName := "test-collection-1"

					collection := conn.Collection(collectionName)
					So(collection, ShouldNotBeNil)

					Convey("which has not yet been fully created in the database", func() {
						cs, err := mongoClient.Database(database).ListCollectionNames(ctx, bson.M{})
						So(err, ShouldBeNil)
						So(cs, ShouldNotContain, collectionName)

						Convey("until an operation has been performed on the collection", func() {
							_, err = collection.InsertOne(ctx, bson.M{"_id": 1})
							So(err, ShouldBeNil)

							cs, err := mongoClient.Database(database).ListCollectionNames(ctx, bson.M{})
							So(err, ShouldBeNil)
							So(cs, ShouldContain, collectionName)
						})
					})
				})
			})

			Convey("For ListCollectionsFor", func() {

				Convey("When called before a collection has been created", func() {
					collectionName := "test-collection-2"

					cs, err := conn.ListCollectionsFor(ctx, database)
					So(err, ShouldBeNil)
					So(cs, ShouldNotContain, collectionName)

					Convey("When called after a collection has been created", func() {
						_, err = conn.Collection(collectionName).InsertOne(ctx, bson.M{"_id": 1})
						So(err, ShouldBeNil)

						cs, err := conn.ListCollectionsFor(ctx, database)
						So(err, ShouldBeNil)
						So(cs, ShouldContain, collectionName)
					})
				})
			})

			Convey("For DropDatabase", func() {

				Convey("before being dropped the database exists", func() {
					dbs, err := mongoClient.ListDatabaseNames(ctx, bson.M{})
					So(err, ShouldBeNil)
					So(dbs, ShouldContain, database)

					Convey("after being dropped does not exist", func() {
						err = conn.DropDatabase(ctx)
						So(err, ShouldBeNil)

						dbs, err := mongoClient.ListDatabaseNames(ctx, bson.M{})
						So(err, ShouldBeNil)
						So(dbs, ShouldNotContain, database)
					})
				})
			})

			Convey("When we run a valid command", func() {
				err = conn.RunCommand(context.Background(), bson.D{
					{Key: "createUser", Value: "test-user-1"},
					{Key: "pwd", Value: "password"},
					{Key: "roles", Value: []bson.M{}}})
				Convey("Then there are no errors", func() {
					So(err, ShouldBeNil)
				})
			})

			Convey("When we run an invalid command", func() {
				err = conn.RunCommand(ctx, bson.D{{Key: "createUser", Value: "test-user-1"}})
				Convey("Then there is an error", func() {
					So(err, ShouldNotBeNil)
				})
			})

			Convey("When Close is called", func() {
				err = conn.Close(ctx)

				Convey("The server closes without error", func() {
					So(err, ShouldBeNil)
					So(conn.Ping(ctx, 10*time.Millisecond), ShouldResemble, errors.New("Failed to ping datastore: client is disconnected"))
				})
			})
		})
	})
}
