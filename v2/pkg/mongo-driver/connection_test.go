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
	connectionConfig := getMongoConnectionConfig()
	if err := checkTcpConnection(connectionConfig.ClusterEndpoint); err != nil {
		log.Event(nil, "mongo db instance not available, skip tests", log.ERROR, log.Error(err))
		t.Skip()
	}

	if mongoConnection, err = mongoDriver.Open(connectionConfig); err != nil {
		log.Event(nil, "mongo instance not available, skip timestamp tests", log.INFO, log.Error(err))
		return
	}

	if err := setUpTestData(mongoConnection); err != nil {
		log.Event(nil, "failed to insert test data, skipping tests", log.ERROR, log.Error(err))
		t.FailNow()
	}

	executeMongoDatesTestSuite(t, mongoConnection)
	executeMongoQueryTestSuite(t, mongoConnection)

	if err := cleanupTestData(mongoConnection); err != nil {
		log.Event(nil, "failed to delete test data", log.ERROR, log.Error(err))
	}
}

func TestSuccessfulMongoDatesViaDocumentDB(t *testing.T) {
	var err error
	var documentDBConnection *mongoDriver.MongoConnection
	connectionConfig := getDocumentDbConnectionConfig()
	if err := checkTcpConnection(connectionConfig.ClusterEndpoint); err != nil {
		log.Event(nil, "documentdb instance not available, skip tests", log.ERROR, log.Error(err))
		t.Skip()
	}
	if documentDBConnection, err = mongoDriver.Open(connectionConfig); err != nil {
		log.Event(nil, "documentdb instance not available, skip timestamp tests", log.INFO, log.Error(err))
		return
	}

	if err := setUpTestData(documentDBConnection); err != nil {
		log.Event(nil, "failed to insert test data, skipping tests", log.ERROR, log.Error(err))
		t.FailNow()
	}

	executeMongoDatesTestSuite(t, documentDBConnection)
	executeMongoQueryTestSuite(t, documentDBConnection)

	if err := cleanupTestData(documentDBConnection); err != nil {
		log.Event(nil, "failed to delete test data", log.ERROR, log.Error(err))
	}
}

func executeMongoDatesTestSuite(t *testing.T, dataStoreConnection *mongoDriver.MongoConnection) {
	Convey("WithUpdates adds both fields", t, func() {

		Convey("check data in original state", func() {

			res := TestModel{}

			err := queryMongo(dataStoreConnection, bson.M{"_id": 1}, &res)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, "first")
		})

		Convey("check data after plain Update", func() {
			res := TestModel{}
			_, err := dataStoreConnection.GetConfiguredCollection().UpdateId(context.Background(), 1, bson.M{"$set": bson.M{"new_key": 123}})
			So(err, ShouldBeNil)

			err = queryMongo(dataStoreConnection, bson.M{"_id": 1}, &res)
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

			_, err = dataStoreConnection.GetConfiguredCollection().UpdateId(context.Background(), 1, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryMongo(dataStoreConnection, bson.M{"_id": 1}, &res)
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

			_, err = dataStoreConnection.GetConfiguredCollection().UpdateId(context.Background(), 1, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryMongo(dataStoreConnection, bson.M{"_id": 1}, &res)
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

func executeMongoQueryTestSuite(t *testing.T, dataStoreConnection *mongoDriver.MongoConnection) {

	Convey("UpsertId should insert if not exists", t, func() {
		_, err := dataStoreConnection.
			GetConfiguredCollection().
			UpsertId(context.Background(), 4, bson.M{"$set": bson.M{"new_key": 456}})
		So(err, ShouldBeNil)

		res := TestModel{}

		err = queryMongo(dataStoreConnection, bson.M{"_id": 4}, &res)
		So(err, ShouldBeNil)
		So(res.NewKey, ShouldEqual, 456)

	})
	Convey("UpsertId should update if  exists", t, func() {
		_, err := dataStoreConnection.
			GetConfiguredCollection().
			UpsertId(context.Background(), 3, bson.M{"$set": bson.M{"new_key": 789}})
		So(err, ShouldBeNil)

		res := TestModel{}
		err = queryMongo(dataStoreConnection, bson.M{"_id": 3}, &res)
		So(err, ShouldBeNil)
		So(res.NewKey, ShouldEqual, 789)
	})

	Convey("UpdateId should update data if document exists", t, func() {
		_, err := dataStoreConnection.GetConfiguredCollection().UpdateId(context.Background(), 3, bson.M{"$set": bson.M{"new_key": 7892}})
		So(err, ShouldBeNil)

		res := TestModel{}

		err = queryMongo(dataStoreConnection, bson.M{"_id": 3}, &res)
		So(err, ShouldBeNil)
		So(res.NewKey, ShouldEqual, 7892)
	})

	Convey("FindOne should find data if document exists", t, func() {
		res := TestModel{}
		err := dataStoreConnection.
			GetConfiguredCollection().
			FindOne(context.Background(), bson.M{"_id": 3}, &res)
		So(err, ShouldBeNil)

		So(res.State, ShouldEqual, "third")
	})

}
func getMongoConnectionConfig() *mongoDriver.MongoConnectionConfig {
	return &mongoDriver.MongoConnectionConfig{
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:                      "test",
		Password:                      "test",
		ClusterEndpoint:               "localhost:27017",
		Database:                      "testDb",
		Collection:                    "testCollection",
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
	}
}

func getDocumentDbConnectionConfig() *mongoDriver.MongoConnectionConfig {
	return &mongoDriver.MongoConnectionConfig{
		CaFilePath:              "./test/data/rds-combined-ca-bundle.pem",
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:                      "test",
		Password:                      "test",
		ClusterEndpoint:               "localhost:27017",
		Database:                      "recipes",
		Collection:                    "recipes",
		SkipCertVerification:          true,
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
	}
}

func setUpTestData(mongoConnection *mongoDriver.MongoConnection) error {
	ctx := context.Background()
	for i, data := range getTestData() {
		if _, err := mongoConnection.
			GetConfiguredCollection().
			UpsertId(ctx, i+1, bson.M{"$set": data}); err != nil {
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
		bson.M{
			"state": "third",
		},
	}
}

func queryMongo(mongoConnection *mongoDriver.MongoConnection, query bson.M, res interface{}) error {
	ctx := context.Background()
	collection := mongoConnection.GetConfiguredCollection()
	if err := collection.FindOne(ctx, query, res); err != nil {
		return err
	}

	return nil
}

func cleanupTestData(connection *mongoDriver.MongoConnection) error {
	return nil
}
