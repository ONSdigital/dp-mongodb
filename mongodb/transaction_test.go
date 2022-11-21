package mongodb_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	"go.mongodb.org/mongo-driver/bson"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	db, collection1, collection2 = "test-db", "test-collection-1", "test-collection-2"
)

// Example of how to use the Connection.RunTransaction() method to perform a series of mongo operations within a transaction
func ExampleTransaction() {
	ctx := context.Background()

	driverConfig := &mongoDriver.MongoDriverConfig{
		ConnectTimeout:                5 * time.Second,
		QueryTimeout:                  5 * time.Second,
		ClusterEndpoint:               "address-of-cluster",
		ReplicaSet:                    "replica-set-name",
		IsWriteConcernMajorityEnabled: true,
		Database:                      db,
		Collections:                   map[string]string{"collection1": collection1, "collection2": collection2},
	}
	conn, err := mongoDriver.Open(driverConfig)
	if err != nil {
		// log error, cannot use mongo db
	}

	r, e := conn.RunTransaction(ctx, true, exampleTransactionFunc(conn, false))

	switch {
	// handle this special case, where we have aborted the transaction because the object was not in a valid state
	case errors.Is(e, badObjectState):

	// otherwise, a runtime error, i.e. couldn't complete the transaction for some other reason (even with retries)
	case !errors.Is(e, nil):

	// transaction completed successfully, and r contains a valid object containing the 2 returned simpleObjects
	default:
		if _, ok := r.(struct{ o1, o2 simpleObject }); !ok {
			// Armageddon!
		}
	}
}

var badObjectState = errors.New("object state incorrect")

// exampleTransactionFunc returns a sample TransactionFunc that executes a series of mongo based operations
// within a transaction defined by the transactionCtx parameter of the function.
// The transactionCtx is derived from the MongoConnection conn
// The interleave boolean value allows injection a mongo call outside of the main transaction in transactionCtx, to
// simulate a parallel transaction that interferes with the main transaction. Without retrying the main transaction,
// this will lead to a Transaction Error on commit
func exampleTransactionFunc(conn *mongoDriver.MongoConnection, interleave bool) mongoDriver.TransactionFunc {
	return func(transactionCtx context.Context) (interface{}, error) {
		var obj1, obj2 simpleObject
		err := conn.Collection(collection1).FindOne(transactionCtx, bson.M{"_id": 1}, &obj1)
		if err != nil {
			return nil, fmt.Errorf("could not find object in collection (%s): %w", collection1, err)
		}

		switch obj1.State {
		case "first":
			obj1.State = "second"
		case "second":
			obj1.State = "third"

			obj2 = simpleObject{ID: 1, State: "final"}
			_, err = conn.Collection(collection2).Upsert(transactionCtx, bson.M{"_id": obj2.ID}, bson.M{"$set": obj2})
			if err != nil {
				return nil, fmt.Errorf("could not upsert object in collection (%s): %w", collection2, err)
			}
		default:
			return nil, badObjectState
		}

		if interleave {
			_, err = conn.Collection(collection1).Update(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"state": "second"}})
			if err != nil {
				return nil, fmt.Errorf("interleave write failed in collection (%s): %w", collection1, err)
			}
		}

		_, err = conn.Collection(collection1).Update(transactionCtx, bson.M{"_id": 1}, bson.M{"$set": obj1})
		if err != nil {
			return nil, fmt.Errorf("could not write object in collection (%s): %w", collection1, err)
		}

		return struct{ o1, o2 simpleObject }{o1: obj1, o2: obj2}, nil
	}
}

func TestTransaction(t *testing.T) {
	ctx := context.Background()
	conn, cleanup := setupMongoConnection(t)
	defer cleanup(ctx)

	Convey("Given a mongo server cluster", t, func() {
		Convey("setup with a test simpleObject in 'first' State in test-collection-1", func() {
			setupTest(t, conn, collection1, simpleObject{ID: 1, State: "first"})

			Convey("when the example transaction is run with neither retries nor an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, false))
				Convey("the transaction completes successfully, amd the results are as expected", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "second"})
					So(res.o2, ShouldResemble, simpleObject{})
				})
			})

			Convey("when the example transaction is run with retries (but without an interleaved outside transaction)", func() {
				r, e := conn.RunTransaction(ctx, true, exampleTransactionFunc(conn, false))
				Convey("again the transaction completes successfully, and the results are the same, as expected", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "second"})
					So(res.o2, ShouldResemble, simpleObject{})
				})
			})

			Convey("when the example transaction is run without retries but with an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, true))
				Convey("the transaction is aborted and fails", func() {
					So(e, ShouldNotBeNil)
					So(r, ShouldBeNil)
				})
			})

			Convey("when the example transaction is run with retries and with an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, true, exampleTransactionFunc(conn, true))
				Convey("the transaction completes successfully, since it is retried and the retry succeeds, but with different results from the above successful completions", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "third"})
					So(res.o2, ShouldResemble, simpleObject{ID: 1, State: "final"})
				})
			})
		})

		Convey("setup with a test simpleObject in 'second' State in test-collection-1", func() {
			setupTest(t, conn, collection1, simpleObject{ID: 1, State: "second"})

			Convey("when the example transaction is run without retries or an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, false))
				Convey("the transaction completes successfully, amd the results are as expected", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "third"})
					So(res.o2, ShouldResemble, simpleObject{ID: 1, State: "final"})
				})
			})

			Convey("when the example transaction is run with retries (but without an interleaved outside transaction), the same results are obtained", func() {
				r, e := conn.RunTransaction(ctx, true, exampleTransactionFunc(conn, false))
				Convey("again the transaction completes successfully, and the results are the same, as expected", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "third"})
					So(res.o2, ShouldResemble, simpleObject{ID: 1, State: "final"})
				})
			})

			Convey("when the example transaction is run without retries but with an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, true))
				Convey("the same results are obtained since the interleaved transaction does not 'interfere' with the main transaction", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "third"})
					So(res.o2, ShouldResemble, simpleObject{ID: 1, State: "final"})
				})
			})

			Convey("when the example transaction is run with retries and with an interleaved outside transaction", func() {
				r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, true))
				Convey("again the same results are obtained since the interleaved transaction does not 'interfere' with the main transaction", func() {
					So(e, ShouldBeNil)
					res, ok := r.(struct{ o1, o2 simpleObject })
					So(ok, ShouldBeTrue)
					So(res.o1, ShouldResemble, simpleObject{ID: 1, State: "third"})
					So(res.o2, ShouldResemble, simpleObject{ID: 1, State: "final"})
				})
			})
		})

		Convey("setup with a test object in 'invalid' State", func() {
			setupTest(t, conn, collection1, simpleObject{ID: 1, State: "invalid"})

			Convey("when the example transaction is run without retries or an interleaved outside transaction", func() {
				Convey("the transaction is explicitly aborted by teh exampleTransactionFunc(), and the expected error and result returned", func() {
					r, e := conn.RunTransaction(ctx, false, exampleTransactionFunc(conn, false))
					So(e, ShouldEqual, badObjectState)
					So(r, ShouldBeNil)
				})
			})

			Convey("and the same results are obtained when the example transaction is run with retries (but without an interleaved outside transaction)", func() {
				r, e := conn.RunTransaction(ctx, true, exampleTransactionFunc(conn, false))
				So(e, ShouldEqual, badObjectState)
				So(r, ShouldBeNil)
			})
		})
	})
}

type simpleObject struct {
	ID    int    `bson:"_id"`
	State string `bson:"state"`
}

func setupMongoConnection(t *testing.T) (*mongoDriver.MongoConnection, func(context.Context)) {
	mongoServer, err := mim.StartWithReplicaSet(context.Background(), "5.0.2", "my-replica-set")
	if err != nil {
		t.Fatalf("failed to start mongo server: %v", err)
	}

	driverConfig := &mongoDriver.MongoDriverConfig{
		ConnectTimeout:                5 * time.Second,
		QueryTimeout:                  5 * time.Second,
		ClusterEndpoint:               mongoServer.URI(),
		ReplicaSet:                    mongoServer.ReplicaSet(),
		Database:                      db,
		Collections:                   map[string]string{collection1: collection1, collection2: collection2},
		IsWriteConcernMajorityEnabled: true,
	}
	conn, err := mongoDriver.Open(driverConfig)
	if err != nil {
		t.Fatalf("couldn't open mongo: %v", err)
	}

	return conn, func(ctx context.Context) { mongoServer.Stop(ctx) }
}

func setupTest(t *testing.T, conn *mongoDriver.MongoConnection, collection string, obj simpleObject) {
	ures, err := conn.Collection(collection).Upsert(context.Background(), bson.M{"_id": obj.ID}, bson.M{"$set": obj})
	if err != nil || (ures.UpsertedCount != 1 && ures.MatchedCount != 1) {
		t.Fatalf("failed to upsert test data into %s", collection)
	}
}
