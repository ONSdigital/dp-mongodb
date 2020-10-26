package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-mongodb/health"
	"github.com/ONSdigital/dp-mongodb/health/mock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	errUnableToConnect          = errors.New("unable to connect with MongoDB")
	errUnableToPingDB           = errors.New("unable to ping DB")
	errorCollectionDoesNotExist = errors.New("collection not found in database")
)

func TestClient_GetOutput(t *testing.T) {

	ctx := context.Background()

	Convey("Given that health endpoint returns 'Success'", t, func() {

		// MongoClient with success healthcheck
		c := &health.CheckMongoClient{
			Client:      *health.NewClient(nil),
			Healthcheck: healthSuccess,
		}

		// CheckState for test validation
		checkState := healthcheck.NewCheckState(health.ServiceName)

		Convey("Checker updates the CheckState to an OK status", func() {
			c.Checker(ctx, checkState)
			So(checkState.Status(), ShouldEqual, healthcheck.StatusOK)
			So(checkState.Message(), ShouldEqual, health.HealthyMessage)
			So(checkState.StatusCode(), ShouldEqual, 0)
		})
	})

	Convey("Given that health endpoint returns 'Failure'", t, func() {

		// MongoClient with failure healthcheck
		c := &health.CheckMongoClient{
			Client:      *health.NewClient(nil),
			Healthcheck: healthFailure,
		}

		// CheckState for test validation
		checkState := healthcheck.NewCheckState(health.ServiceName)

		Convey("Checker updates the CheckState to a CRITICAL status", func() {
			c.Checker(ctx, checkState)
			So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
			So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
			So(checkState.StatusCode(), ShouldEqual, 0)
		})
	})
}

func createSessionMocks() (*mock.SessionMock, *mock.SessionMock, *mock.DataLayerMock) {
	mockedDatabaser := &mock.DataLayerMock{
		CollectionNamesFunc: func() ([]string, error) {
			return []string{"collectionOne"}, nil
		},
	}

	copiedSession := &mock.SessionMock{
		CloseFunc: func() {},
		PingFunc: func() error {
			return nil
		},
		DBFunc: func(string) health.DataLayer {
			return mockedDatabaser
		},
	}

	mainSession := &mock.SessionMock{
		CopyFunc: func() health.Session {
			return copiedSession
		},
	}

	return mainSession, copiedSession, mockedDatabaser
}

func createSessionMocksMultipleCollections() (*mock.SessionMock, *mock.SessionMock, *mock.DataLayerMock) {
	mockedDatabaser := &mock.DataLayerMock{
		CollectionNamesFunc: func() ([]string, error) {
			return []string{"collectionOne", "collectionTwo", "collectionThree", "collectionFour"}, nil
		},
	}

	copiedSession := &mock.SessionMock{
		CloseFunc: func() {},
		PingFunc: func() error {
			return nil
		},
		DBFunc: func(string) health.DataLayer {
			return mockedDatabaser
		},
	}

	mainSession := &mock.SessionMock{
		CopyFunc: func() health.Session {
			return copiedSession
		},
	}

	return mainSession, copiedSession, mockedDatabaser
}

func createSessionMocksCollectionNamesError() (*mock.SessionMock, *mock.SessionMock, *mock.DataLayerMock) {
	mockedDatabaser := &mock.DataLayerMock{
		CollectionNamesFunc: func() ([]string, error) {
			return nil, errUnableToConnect
		},
	}

	copiedSession := &mock.SessionMock{
		CloseFunc: func() {},
		PingFunc: func() error {
			return nil
		},
		DBFunc: func(string) health.DataLayer {
			return mockedDatabaser
		},
	}

	mainSession := &mock.SessionMock{
		CopyFunc: func() health.Session {
			return copiedSession
		},
	}

	return mainSession, copiedSession, mockedDatabaser
}

func createSessionMocksPingError() (*mock.SessionMock, *mock.SessionMock) {
	copiedSession := &mock.SessionMock{
		CloseFunc: func() {},
		PingFunc: func() error {
			return errUnableToPingDB
		},
	}

	mainSession := &mock.SessionMock{
		CopyFunc: func() health.Session {
			return copiedSession
		},
	}

	return mainSession, copiedSession
}

func TestClient_Healthcheck(t *testing.T) {

	ctx := context.Background()

	Convey("Given that the databaseCollection is nil", t, func() {

		main, copied, _ := createSessionMocks()
		c := health.NewClient(main)

		Convey("Healthcheck returns the serviceName and nil error, and the database isn't called", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldEqual, nil)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 0)
		})

	})

	Convey("Given that the databaseCollection has one database but the collection list is empty", t, func() {

		dc := make(map[health.Database][]health.Collection)
		dc["databaseOne"] = []health.Collection{}
		main, copied, _ := createSessionMocks()
		c := health.NewClientWithCollections(main, dc)

		Convey("Healthcheck returns the ServiceName and nil error, and the database is called once", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldEqual, nil)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 1)
		})

	})

	Convey("Given that the databaseCollection has one database and one collection and the collection exists", t, func() {

		dc := make(map[health.Database][]health.Collection)
		dc["databaseOne"] = []health.Collection{"collectionOne"}
		main, copied, _ := createSessionMocks()
		c := health.NewClientWithCollections(main, dc)

		Convey("Healthcheck returns the serviceName and nil error, and the database is called once", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldEqual, nil)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 1)
		})
	})

	Convey("Given that the databaseCollection has one database and one collection and the collection does not exist", t, func() {

		dc := make(map[health.Database][]health.Collection)
		dc["databaseOne"] = []health.Collection{"collectionTwo"}
		main, copied, _ := createSessionMocks()
		c := health.NewClientWithCollections(main, dc)

		Convey("Healthcheck returns the serviceName and an error, and the database is called once", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldNotBeNil)
			So(err, ShouldResemble, errorCollectionDoesNotExist)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 1)
		})
	})

	Convey("Given that the databaseCollection has two databases and four collections in each one and the collections exist", t, func() {

		dc := make(map[health.Database][]health.Collection)
		dc["databaseOne"] = []health.Collection{"collectionOne", "collectionTwo", "collectionThree", "collectionFour"}
		dc["databaseTwo"] = []health.Collection{"collectionOne", "collectionTwo", "collectionThree", "collectionFour"}
		main, copied, _ := createSessionMocksMultipleCollections()
		c := health.NewClientWithCollections(main, dc)

		Convey("Healthcheck returns the serviceName and nil error, and the database is called twice", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldBeNil)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 2)
		})
	})

	Convey("Given that the call to the Mongo database fails on the collectionNames call", t, func() {

		dc := make(map[health.Database][]health.Collection)
		dc["databaseOne"] = []health.Collection{"collectionOne"}
		main, copied, _ := createSessionMocksCollectionNamesError()
		c := health.NewClientWithCollections(main, dc)

		Convey("Healthcheck returns the serviceName and error, and the database is called once", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldNotBeNil)
			So(err, ShouldResemble, errUnableToConnect)
			So(res, ShouldEqual, "mongodb")
			So(copied.DBCalls(), ShouldHaveLength, 1)
		})
	})

	Convey("Given that the ping to the mongo client returns an error", t, func() {
		main, _ := createSessionMocksPingError()
		c := health.NewClient(main)

		Convey("Healthcheck returns the serviceName and error", func() {
			res, err := c.Healthcheck(ctx)
			So(err, ShouldNotBeNil)
			So(err, ShouldResemble, errUnableToPingDB)
			So(res, ShouldEqual, "mongodb")
		})
	})

}

var (
	healthSuccess = func(context.Context) (string, error) {
		return "Success", nil
	}

	healthFailure = func(context.Context) (string, error) {
		return "Failure", errUnableToConnect
	}
)
