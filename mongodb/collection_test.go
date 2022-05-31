package mongodb_test

import (
	"context"
	"testing"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	. "github.com/smartystreets/goconvey/convey"
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
	})
}
