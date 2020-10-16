package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-mongodb/health"
	. "github.com/smartystreets/goconvey/convey"
)

var errUnableToConnect = errors.New("unable to connect to mongo datastore")

func TestClient_GetOutput(t *testing.T) {

	ctx := context.Background()

	dc1 := make(map[health.Database][]health.Collection)
	dc1["databaseOne"] = []health.Collection{"collectionOne"}

	Convey("Given that health endpoint returns 'Success'", t, func() {

		// MongoClient with success healthcheck
		c := &health.CheckMongoClient{
			Client:      *health.NewClient(nil, dc1),
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
			Client:      *health.NewClient(nil, dc1),
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

var (
	healthSuccess = func(context.Context) (string, error) {
		return "Success", nil
	}

	healthFailure = func(context.Context) (string, error) {
		return "Failure", errUnableToConnect
	}
)
