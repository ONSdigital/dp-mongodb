package mongodb

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSuccessfulMongoDates(t *testing.T) {

	Convey("ensure adds all requested time fields", t, func() {

		now := time.Now()

		timestamp := primitive.Timestamp{T: uint32(now.Unix()), I: 1234}
		anotherTimestamp := primitive.Timestamp{T: uint32(now.Unix()), I: 1235}

		Convey("check WithUniqueTimestampQuery", func() {

			query := bson.M{"foo": bson.M{"bar": 321}}
			queryWithTimestamp := WithUniqueTimestampQuery(query, timestamp)
			So(queryWithTimestamp, ShouldResemble, bson.M{"unique_timestamp": timestamp, "foo": bson.M{"bar": 321}})

		})

		Convey("check WithNamespacedUniqueTimestampQuery", func() {

			query := bson.M{"foo": bson.M{"key": 12345}}
			queryWithTimestamps := WithNamespacedUniqueTimestampQuery(query, []primitive.Timestamp{timestamp, anotherTimestamp}, []string{"nixed.", "currant."})
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
