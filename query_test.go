package mongo

import (
	"context"
	"errors"
	"testing"
	"time"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/ONSdigital/log.go/v2/log"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	returnContextKey = "want_return"
	early            = "early"
)

type ungraceful struct{}

func (t ungraceful) shutdown(ctx context.Context, session *mgo.Session, closedChannel chan bool) {
	time.Sleep(timeLeft + (100 * time.Millisecond))
	if ctx.Value(returnContextKey) == early || ctx.Err() != nil {
		return
	}

	session.Close()

	closedChannel <- true
	return
}

var (
	hasSessionSleep bool
	session         *mgo.Session

	Collection = "test"
	Database   = "test"
	URI        = "localhost:27017"
)

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Collection string
	Database   string
	URI        string
}

type TestModel struct {
	State           string               `bson:"state"`
	NewKey          int                  `bson:"new_key,omitempty"`
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *bson.MongoTimestamp `bson:"unique_timestamp,omitempty"`
}

type Times struct {
	LastUpdated     time.Time            `bson:"last_updated"`
	UniqueTimestamp *bson.MongoTimestamp `bson:"unique_timestamp,omitempty"`
}

type testNamespacedModel struct {
	State   string `bson:"state"`
	NewKey  int    `bson:"new_key,omitempty"`
	Currant Times  `bson:"currant,omitempty"`
	Nixed   Times  `bson:"nixed,omitempty"`
}

func TestSuccessfulMongoDates(t *testing.T) {

	Convey("ensure adds all requested time fields", t, func() {

		now := time.Now()
		timestamp, err := bson.NewMongoTimestamp(now, 1234)
		So(err, ShouldBeNil)
		anotherTimestamp, err := bson.NewMongoTimestamp(now, 1235)
		So(err, ShouldBeNil)

		Convey("check WithUniqueTimestampQuery", func() {

			query := bson.M{"foo": bson.M{"bar": 321}}
			queryWithTimestamp := WithUniqueTimestampQuery(query, timestamp)
			So(queryWithTimestamp, ShouldResemble, bson.M{"unique_timestamp": timestamp, "foo": bson.M{"bar": 321}})

		})

		Convey("check WithNamespacedUniqueTimestampQuery", func() {

			query := bson.M{"foo": bson.M{"key": 12345}}
			queryWithTimestamps := WithNamespacedUniqueTimestampQuery(query, []bson.MongoTimestamp{timestamp, anotherTimestamp}, []string{"nixed.", "currant."})
			So(queryWithTimestamps, ShouldResemble, bson.M{
				"currant.unique_timestamp": anotherTimestamp,
				"nixed.unique_timestamp":   timestamp,
				"foo":                      bson.M{"key": 12345},
			})

		})

		Convey("check WithUpdates", func() {

			update := bson.M{"$set": bson.M{"new_key": 321}}
			updateWithTimestamps, err := WithUpdates(update)
			So(err, ShouldBeNil)
			So(updateWithTimestamps, ShouldResemble, bson.M{
				"$currentDate": bson.M{
					"last_updated":     true,
					"unique_timestamp": bson.M{"$type": "timestamp"},
				},
				"$set": bson.M{"new_key": 321},
			})

		})

		Convey("check WithNamespacedUpdates", func() {

			update := bson.M{"$set": bson.M{"new_key": 1234}}
			updateWithTimestamps, err := WithNamespacedUpdates(update, []string{"nixed.", "currant."})
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

		})

	})
}

func TestSuccessfulMongoDatesViaMongo(t *testing.T) {
	session = nil
	if _, err := setupSession(); err != nil {
		log.Info(context.Background(), "mongo instance not available, skip timestamp tests", log.FormatErrors([]error{err}))
		return
	}

	if err := setUpTestData(session.Copy()); err != nil {
		log.Error(context.Background(), "failed to insert test data, skipping tests", err)
		t.FailNow()
	}

	Convey("WithUpdates adds both fields", t, func() {

		Convey("check data in original state", func() {

			res := TestModel{}

			err := queryMongo(session.Copy(), bson.M{"_id": "1"}, &res)
			So(err, ShouldBeNil)
			So(res, ShouldResemble, TestModel{State: "first"})

		})

		Convey("check data after plain Update", func() {

			res := TestModel{}

			err := session.DB(Database).C(Collection).Update(bson.M{"_id": "1"}, bson.M{"$set": bson.M{"new_key": 123}})
			So(err, ShouldBeNil)

			err = queryMongo(session.Copy(), bson.M{"_id": "1"}, &res)
			So(err, ShouldBeNil)
			So(res, ShouldResemble, TestModel{State: "first", NewKey: 123})

		})

		Convey("check data with Update with new dates", func() {

			testStartTime := time.Now().Truncate(time.Second)
			res := TestModel{}

			update := bson.M{"$set": bson.M{"new_key": 321}}
			updateWithTimestamps, err := WithUpdates(update)
			So(err, ShouldBeNil)
			So(updateWithTimestamps, ShouldResemble, bson.M{"$currentDate": bson.M{"last_updated": true, "unique_timestamp": bson.M{"$type": "timestamp"}}, "$set": bson.M{"new_key": 321}})

			err = session.DB(Database).C(Collection).Update(bson.M{"_id": "1"}, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryMongo(session.Copy(), bson.M{"_id": "1"}, &res)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, "first")
			So(res.NewKey, ShouldEqual, 321)
			So(res.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
			// extract time part
			So(res.UniqueTimestamp.Time(), ShouldHappenOnOrAfter, testStartTime)

		})

		Convey("check data with Update with new Namespaced dates", func() {

			// ensure this testStartTime is greater than last
			time.Sleep(1010 * time.Millisecond)
			testStartTime := time.Now().Truncate(time.Second)
			res := testNamespacedModel{}

			update := bson.M{"$set": bson.M{"new_key": 1234}}
			updateWithTimestamps, err := WithNamespacedUpdates(update, []string{"nixed.", "currant."})
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

			err = session.DB(Database).C(Collection).Update(bson.M{"_id": "1"}, updateWithTimestamps)
			So(err, ShouldBeNil)

			err = queryNamespacedMongo(session.Copy(), bson.M{"_id": "1"}, &res)
			So(err, ShouldBeNil)
			So(res.State, ShouldEqual, "first")
			So(res.NewKey, ShouldEqual, 1234)
			So(res.Currant.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
			So(res.Nixed.LastUpdated, ShouldHappenOnOrAfter, testStartTime)
			// extract time part
			So(res.Currant.UniqueTimestamp.Time(), ShouldHappenOnOrAfter, testStartTime)
			So(res.Nixed.UniqueTimestamp.Time(), ShouldHappenOnOrAfter, testStartTime)

		})

	})

	if err := cleanupTestData(session.Copy()); err != nil {
		log.Error(context.Background(), "failed to delete test data", err)
	}
}

func cleanupTestData(session *mgo.Session) error {
	defer session.Close()

	err := session.DB(Database).DropDatabase()
	if err != nil {
		return err
	}

	return nil
}

func slowQueryMongo(session *mgo.Session) error {
	defer session.Close()

	_, err := session.DB(Database).C(Collection).Find(bson.M{"$where": "sleep(2000) || true"}).Count()
	if err != nil {
		return err
	}

	return nil
}

func queryMongo(session *mgo.Session, query bson.M, res *TestModel) error {
	defer session.Close()

	if err := session.DB(Database).C(Collection).Find(query).One(&res); err != nil {
		return err
	}

	return nil
}

func queryNamespacedMongo(session *mgo.Session, query bson.M, res *testNamespacedModel) error {
	defer session.Close()

	if err := session.DB(Database).C(Collection).Find(query).One(&res); err != nil {
		return err
	}

	return nil
}

func getTestData() []bson.M {
	return []bson.M{
		{
			"_id":   "1",
			"state": "first",
		},
		{
			"_id":   "2",
			"state": "second",
		},
	}
}

func setUpTestData(session *mgo.Session) error {
	defer session.Close()

	if _, err := session.DB(Database).C(Collection).Upsert(bson.M{"_id": "1"}, getTestData()[0]); err != nil {
		return err
	}

	if _, err := session.DB(Database).C(Collection).Upsert(bson.M{"_id": "2"}, getTestData()[1]); err != nil {
		return err
	}

	return nil
}

func setupSession() (*Mongo, error) {
	mongo := &Mongo{
		Collection: Collection,
		Database:   Database,
		URI:        URI,
	}

	if session != nil {
		return nil, errors.New("failed to initialise mongo")
	}

	var err error

	if session, err = mgo.Dial(URI); err != nil {
		return nil, err
	}

	session.EnsureSafe(&mgo.Safe{WMode: "majority"})
	session.SetMode(mgo.Strong, true)
	return mongo, nil
}
