package health

import (
	"context"
	"errors"
	"testing"
	"time"

	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	. "github.com/smartystreets/goconvey/convey"
)

var errUnableToConnect = errors.New("unable to connect to mongo datastore")

func TestClient_GetOutput(t *testing.T) {
	defaultTime := time.Now().UTC()
	ctx := context.Background()
	apiName := "test-service"

	Convey("Given that health endpoint returns 'Success'", t, func() {
		c := &CheckMongoClient{
			client: Client{
				serviceName: apiName,
				Check:       &health.Check{Name: apiName},
			},
			healthcheck: healthSuccess,
		}

		Convey("Checker returns a status OK Check structure", func() {
			check, err := c.Checker(ctx)
			So(check, ShouldResemble, c.client.Check)
			So(check.Name, ShouldEqual, apiName)
			So(check.StatusCode, ShouldEqual, 0)
			So(check.Status, ShouldEqual, health.StatusOK)
			So(check.Message, ShouldEqual, healthyMessage)
			So(*check.LastChecked, ShouldHappenAfter, defaultTime)
			So(check.LastFailure, ShouldBeNil)
			So(*check.LastSuccess, ShouldHappenAfter, defaultTime)
			So(err, ShouldBeNil)
		})
	})

	Convey("Given that health endpoint returns 'Failure'", t, func() {
		c := &CheckMongoClient{
			client: Client{
				serviceName: apiName,
				Check:       &health.Check{Name: apiName},
			},
			healthcheck: healthFailure,
		}

		Convey("Checker returns a status CRITICAL Check structure", func() {
			check, err := c.Checker(ctx)
			So(check, ShouldResemble, c.client.Check)
			So(check.Name, ShouldEqual, apiName)
			So(check.StatusCode, ShouldEqual, 0)
			So(check.Status, ShouldEqual, health.StatusCritical)
			So(check.Message, ShouldEqual, errUnableToConnect.Error())
			So(*check.LastChecked, ShouldHappenAfter, defaultTime)
			So(*check.LastFailure, ShouldHappenAfter, defaultTime)
			So(check.LastSuccess, ShouldBeNil)
			So(err, ShouldResemble, errUnableToConnect)
		})
	})
}

func TestCheckerHistory(t *testing.T) {

	ctx := context.Background()
	apiName := "test-service"

	Convey("Given that we have a mongo client with previous successful checks", t, func() {
		lastCheckTime := time.Now().UTC().Add(1 * time.Minute)
		previousCheck := createSuccessfulCheck(lastCheckTime, "healthy", apiName)
		c := &CheckMongoClient{
			client: Client{
				serviceName: apiName,
				Check:       &previousCheck,
			},
			healthcheck: healthFailure,
		}

		Convey("A new healthcheck keeps the non-overwritten values", func() {
			check, _ := c.Checker(ctx)
			So(check.LastSuccess, ShouldResemble, &lastCheckTime)
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

// create a successful check without lastFailed value
func createSuccessfulCheck(t time.Time, msg string, serviceName string) health.Check {
	return health.Check{
		Name:        serviceName,
		LastSuccess: &t,
		LastChecked: &t,
		Status:      health.StatusOK,
		Message:     msg,
	}
}
