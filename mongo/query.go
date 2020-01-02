package mongo

import (
	"errors"
	"time"

	"github.com/globalsign/mgo/bson"
)

// keep these in sync with Timestamps tags below
const (
	LastUpdatedKey     = "last_updated"
	UniqueTimestampKey = "unique_timestamp"
)

// Timestamps represent an object containing time stamps
// keep these in sync with above const
type Timestamps struct {
	LastUpdated     time.Time            `bson:"last_updated,omitempty"     json:"last_updated,omitempty"`
	UniqueTimestamp *bson.MongoTimestamp `bson:"unique_timestamp,omitempty" json:"-"`
}

// withCurrentDate creates or adds $currentDate to updateDoc - populates that with key:val
func withCurrentDate(updateDoc bson.M, key string, val interface{}) (bson.M, error) {
	var currentDate bson.M
	var ok bool
	if currentDate, ok = updateDoc["$currentDate"].(bson.M); !ok {
		currentDate = bson.M{}
	}
	switch v := val.(type) {
	case bool, bson.M:
		currentDate[key] = v
	default:
		return nil, errors.New("withCurrentDate: Cannot handle that type")
	}
	updateDoc["$currentDate"] = currentDate
	return updateDoc, nil
}

// WithUpdates adds all timestamps to updateDoc
func WithUpdates(updateDoc bson.M) (bson.M, error) {
	newUpdateDoc, err := WithLastUpdatedUpdate(updateDoc)
	if err != nil {
		return nil, err
	}
	return WithUniqueTimestampUpdate(newUpdateDoc)
}

// WithNamespacedUpdates adds all timestamps to updateDoc
func WithNamespacedUpdates(updateDoc bson.M, prefixes []string) (bson.M, error) {
	newUpdateDoc, err := WithNamespacedLastUpdatedUpdate(updateDoc, prefixes)
	if err != nil {
		return nil, err
	}
	return WithNamespacedUniqueTimestampUpdate(newUpdateDoc, prefixes)
}

// WithLastUpdatedUpdate adds last_updated to updateDoc
func WithLastUpdatedUpdate(updateDoc bson.M) (bson.M, error) {
	return withCurrentDate(updateDoc, LastUpdatedKey, true)
}

// WithNamespacedLastUpdatedUpdate adds unique timestamp to updateDoc
func WithNamespacedLastUpdatedUpdate(updateDoc bson.M, prefixes []string) (newUpdateDoc bson.M, err error) {
	newUpdateDoc = updateDoc
	for _, prefix := range prefixes {
		if newUpdateDoc, err = withCurrentDate(newUpdateDoc, prefix+LastUpdatedKey, true); err != nil {
			return nil, err
		}
	}
	return newUpdateDoc, nil
}

// WithUniqueTimestampUpdate adds unique timestamp to updateDoc
func WithUniqueTimestampUpdate(updateDoc bson.M) (bson.M, error) {
	return withCurrentDate(updateDoc, UniqueTimestampKey, bson.M{"$type": "timestamp"})
}

// WithNamespacedUniqueTimestampUpdate adds unique timestamp to updateDoc
func WithNamespacedUniqueTimestampUpdate(updateDoc bson.M, prefixes []string) (newUpdateDoc bson.M, err error) {
	newUpdateDoc = updateDoc
	for _, prefix := range prefixes {
		if newUpdateDoc, err = withCurrentDate(newUpdateDoc, prefix+UniqueTimestampKey, bson.M{"$type": "timestamp"}); err != nil {
			return nil, err
		}
	}
	return newUpdateDoc, nil
}

// WithUniqueTimestampQuery adds unique timestamp to queryDoc
func WithUniqueTimestampQuery(queryDoc bson.M, timestamp bson.MongoTimestamp) bson.M {
	queryDoc[UniqueTimestampKey] = timestamp
	return queryDoc
}

// WithNamespacedUniqueTimestampQuery adds unique timestamps to queryDoc sub-docs
func WithNamespacedUniqueTimestampQuery(queryDoc bson.M, timestamps []bson.MongoTimestamp, prefixes []string) bson.M {
	newQueryDoc := queryDoc
	for idx, prefix := range prefixes {
		newQueryDoc[prefix+UniqueTimestampKey] = timestamps[idx]
	}
	return newQueryDoc
}
