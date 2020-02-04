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

		// mock CheckState for test validation
		mockCheckState := mock.CheckStateMock{
			UpdateFunc: func(status, message string, statusCode int) error {
				return nil
			},
		}

		Convey("Checker updates the CheckState to an OK status", func() {
			c.Checker(ctx, &mockCheckState)
			updateCalls := mockCheckState.UpdateCalls()
			So(len(updateCalls), ShouldEqual, 1)
			So(updateCalls[0].Status, ShouldEqual, healthcheck.StatusOK)
			So(updateCalls[0].Message, ShouldEqual, health.HealthyMessage)
			So(updateCalls[0].StatusCode, ShouldEqual, 0)
		})
	})

	Convey("Given that health endpoint returns 'Failure'", t, func() {

		// MongoClient with failure healthcheck
		c := &health.CheckMongoClient{
			Client:      *health.NewClient(nil),
			Healthcheck: healthFailure,
		}

		// mock CheckState for test validation
		mockCheckState := mock.CheckStateMock{
			UpdateFunc: func(status, message string, statusCode int) error {
				return nil
			},
		}

		Convey("Checker updates the CheckState to a CRITICAL status", func() {
			c.Checker(ctx, &mockCheckState)
			updateCalls := mockCheckState.UpdateCalls()
			So(len(updateCalls), ShouldEqual, 1)
			So(updateCalls[0].Status, ShouldEqual, healthcheck.StatusCritical)
			So(updateCalls[0].Message, ShouldEqual, errUnableToConnect.Error())
			So(updateCalls[0].StatusCode, ShouldEqual, 0)
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
