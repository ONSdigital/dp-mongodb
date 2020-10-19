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

var errUnableToConnect = errors.New("unable to connect to mongo datastore")

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

func TestClient_Healthcheck(t *testing.T) {

	ctx := context.Background()

	dc1 := make(map[health.Database][]health.Collection)
	dc1["databaseOne"] = []health.Collection{"collectionOne"}

	mockedDatabaser := &mock.DatabaserMock{
		CollectionNamesFunc: func() ([]string, error) {
			return []string{"collectionOne"}, nil
		},
	}

	copiedSessioner := &mock.SessionerMock{
		CloseFunc: func() {},
		PingFunc: func() error {
			return nil
		},
		DBFunc: func(string) health.Databaser {
			return mockedDatabaser
		},
	}

	mainSessioner := &mock.SessionerMock{
		CopyFunc: func() health.Sessioner {
			return copiedSessioner
		},
	}

	Convey("Given that the databaseCollection is nil", t, func() {

		c := health.NewClient(mainSessioner)

		Convey("Healthcheck returns the serviceName and nil error, and the database isn't called", func() {
			res, err := c.Healthcheck(ctx)
			So(res, ShouldEqual, "mongodb")
			So(err, ShouldEqual, nil)
			So(copiedSessioner.DBCalls(), ShouldHaveLength, 0)
		})

	})

	Convey("Given that the databaseCollection has one database and one collection and the collection exists", t, func() {
		c := health.NewClientWithCollections(mainSessioner, dc1)

		Convey("Healthcheck returns the serviceName and nil error, and the database is called once", func() {
			res, err := c.Healthcheck(ctx)
			So(res, ShouldEqual, "mongodb")
			So(err, ShouldEqual, nil)
			So(copiedSessioner.DBCalls(), ShouldHaveLength, 1)
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
