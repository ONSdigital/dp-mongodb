package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	. "github.com/smartystreets/goconvey/convey"
)

var errUnableToConnect = errors.New("unable to connect to mongo datastore")

func TestHealth_GetCheck(t *testing.T) {
	defaultTime := time.Now().UTC()
	ctx := context.Background()

	Convey("Given an ok state return OK health check object", t, func() {
		check := getCheck(ctx, "mongo", healthcheck.StatusOK)

		So(check.Name, ShouldEqual, "mongo")
		So(check.StatusCode, ShouldEqual, 0)
		So(check.Status, ShouldEqual, health.StatusOK)
		So(check.Message, ShouldEqual, statusDescription[health.StatusOK])
		So(check.LastChecked, ShouldHappenAfter, defaultTime)
		So(check.LastSuccess, ShouldHappenAfter, defaultTime)
		So(check.LastFailure, ShouldEqual, unixTime)
	})

	Convey("Given a critical state return CRITICAL health check object", t, func() {
		check := getCheck(ctx, "mongo", healthcheck.StatusCritical)

		So(check.Name, ShouldEqual, "mongo")
		So(check.StatusCode, ShouldEqual, 0)
		So(check.Status, ShouldEqual, health.StatusCritical)
		So(check.Message, ShouldEqual, statusDescription[health.StatusCritical])
		So(check.LastChecked, ShouldHappenAfter, defaultTime)
		So(check.LastSuccess, ShouldEqual, unixTime)
		So(check.LastFailure, ShouldHappenAfter, defaultTime)
	})
}

func TestClient_GetOutput(t *testing.T) {
	defaultTime := time.Now().UTC()
	ctx := context.Background()
	apiName := "test-service"

	Convey("When health endpoint returns status OK", t, func() {
		c := &CheckMongoClient{
			client: Client{
				serviceName: apiName,
			},
			healthcheck: healthSuccess,
		}

		check, err := c.Checker(ctx)
		So(check.Name, ShouldEqual, apiName)
		So(check.StatusCode, ShouldEqual, 0)
		So(check.Status, ShouldEqual, health.StatusOK)
		So(check.Message, ShouldEqual, statusDescription[health.StatusOK])
		So(check.LastChecked, ShouldHappenAfter, defaultTime)
		So(check.LastFailure, ShouldEqual, unixTime)
		So(check.LastSuccess, ShouldHappenAfter, defaultTime)
		So(err, ShouldBeNil)
	})

	Convey("When health endpoint returns status Critical", t, func() {
		c := &CheckMongoClient{
			client: Client{
				serviceName: apiName,
			},
			healthcheck: healthFailure,
		}

		check, err := c.Checker(ctx)
		So(check.Name, ShouldEqual, apiName)
		So(check.StatusCode, ShouldEqual, 0)
		So(check.Status, ShouldEqual, health.StatusCritical)
		So(check.Message, ShouldEqual, statusDescription[health.StatusCritical])
		So(check.LastChecked, ShouldHappenAfter, defaultTime)
		So(check.LastFailure, ShouldHappenAfter, defaultTime)
		So(check.LastSuccess, ShouldEqual, unixTime)
		So(err, ShouldResemble, errUnableToConnect)
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
