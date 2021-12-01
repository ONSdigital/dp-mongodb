package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMongoCollection(t *testing.T) {
	Convey("Given a mongodb test server is setup and running", t, func() {

		var (
			mongoVersion = "4.4.8"
			user         = "test-user"
			password     = "test-password"
			db           = "test-db"
			collection   = "test-collection"
		)

		mongoServer, err := mim.Start(mongoVersion)
		if err != nil {
			t.Fatalf("failed to start mongo server: %v", err)
		}
		defer mongoServer.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoServer.URI()))
		if err != nil {
			t.Fatalf("failed to connect to mongo server: %v", err)
		}

		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err = client.Database(db).RunCommand(ctx, bson.D{{"createUser", user}, {"pwd", password}, {"roles", []bson.M{}}}).Err()
		if err != nil {
			t.Fatalf("couldn't set up test: %v", err)
		}

		conn, err := Open(&MongoConnectionConfig{
			ConnectTimeoutInSeconds: 5,
			QueryTimeoutInSeconds:   5,

			Username:        user,
			Password:        password,
			ClusterEndpoint: fmt.Sprintf("localhost:%d", mongoServer.Port()),
			Database:        db,
			Collection:      collection,
		})

		type doc struct {
			ID      string `bson:"_id"`
			Content string
		}

		testDoc := doc{ID: "test doc", Content: "test doc contents"}
		Convey("when I insert a test document into the test collection via the mongoDB library Collection", func() {
			result, err := conn.GetConfiguredCollection().Insert(ctx, testDoc)
			So(err, ShouldBeNil)
			So(result.InsertedId, ShouldEqual, testDoc.ID)

			Convey("I can retrieve it with via underlying Mongo collection", func() {
				var d doc
				err = conn.GetConfiguredCollection().GetMongoCollection().FindOne(ctx, bson.M{"_id": testDoc.ID}).Decode(&d)
				So(err, ShouldBeNil)
				So(d, ShouldResemble, testDoc)
			})
		})
	})
}
