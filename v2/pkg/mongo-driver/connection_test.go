package mongo_driver_test

import (
	"context"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v2/pkg/mongo-driver"
	"github.com/ONSdigital/log.go/log"
	. "github.com/smartystreets/goconvey/convey"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
	"time"
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

func TestSuccessfulMongoDatesViaMongo(t *testing.T) {
	var err error
	var mongoConnection *mongoDriver.MongoConnection
	if mongoConnection, err = mongoDriver.Open(getMongoConnectionConfig()); err != nil {
		log.Event(nil, "mongo instance not available, skip timestamp tests", log.INFO, log.Error(err))
		return
	}

	if err := setUpTestData(mongoConnection); err != nil {
		log.Event(nil, "failed to insert test data, skipping tests", log.ERROR, log.Error(err))
		t.FailNow()
	}

	executeTestSuite(t, mongoConnection)

	if err := cleanupTestData(mongoConnection); err != nil {
		log.Event(nil, "failed to delete test data", log.ERROR, log.Error(err))
	}
}

func executeTestSuite(t *testing.T, mongoConnection *mongoDriver.MongoConnection) {
	Convey("WithUpdates adds both fields", t, func() {

		Convey("check data in original state", func() {

			res := TestModel{}

			err := queryMongo(mongoConnection, bson.M{"_id": 1}, &res)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, "first")
		})

		Convey("check data after plain Update", func() {
			res := TestModel{}
			err := mongoConnection.UpdateId(context.Background(), 1, bson.M{"$set": bson.M{"new_key": 123}})
			So(err, ShouldBeNil)

			err = queryMongo(mongoConnection, bson.M{"_id": 1}, &res)
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

			err = mongoConnection.UpdateId(context.Background(), 1, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryMongo(mongoConnection, bson.M{"_id": 1}, &res)
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

			err = mongoConnection.UpdateId(context.Background(), 1, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryMongo(mongoConnection, bson.M{"_id": 1}, &res)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, "first")
			So(res.NewKey, ShouldEqual, 1234)
			So(res.Currant.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
			So(res.Nixed.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
			// extract time part

			So(time.Unix(int64(res.Currant.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)
			So(time.Unix(int64(res.Nixed.UniqueTimestamp.T), 0), ShouldHappenOnOrAfter, testStartTime)

		})

	})
}

func getMongoConnectionConfig() *mongoDriver.MongoConnectionConfig {
	return &mongoDriver.MongoConnectionConfig{
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:        "test",
		Password:        "test",
		ClusterEndpoint: "localhost:27017",
		Database:        "testDb",
		Collection:      "testCollection",
	}
}

func setUpTestData(mongoConnection *mongoDriver.MongoConnection) error {
	ctx := context.Background()
	for i, data := range getTestData() {
		if _, err := mongoConnection.UpsertId(ctx, i+1, bson.M{"$set": data}); err != nil {
			return err
		}
	}
	return nil
}

func getTestData() []bson.M {
	return []bson.M{
		bson.M{
			"state": "first",
		},
		bson.M{
			"state": "second",
		},
	}
}

func queryMongo(mongoConnection *mongoDriver.MongoConnection, query bson.M, res interface{}) error {
	ctx := context.Background()
	if err := mongoConnection.FindOne(ctx, query, res); err != nil {
		return err
	}

	return nil
}

func cleanupTestData(connection *mongoDriver.MongoConnection) error {
	return nil
}
