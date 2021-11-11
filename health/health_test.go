package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-mongodb/v3/health"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	errUnableToConnect       = errors.New("failed to connect")
	errCollectionsDoNotExist = errors.New("can't find collection")
)

func TestClient_GetOutput(t *testing.T) {

	ctx := context.Background()

	Convey("Given a CheckMongoClient without collections", t, func() {
		c := &health.CheckMongoClient{
			Client: *health.NewClient(nil),
		}
		// CheckState for test validation
		checkState := healthcheck.NewCheckState(health.ServiceName)

		Convey("When that health endpoint returns 'Success'", func() {
			c.Healthcheck = healthSuccess
			Convey("Then Checker updates the CheckState to an OK status", func() {
				c.Checker(ctx, checkState)
				So(checkState.Status(), ShouldEqual, healthcheck.StatusOK)
				So(checkState.Message(), ShouldEqual, "mongodb is OK")
				So(checkState.StatusCode(), ShouldEqual, 0)
			})
		})

		Convey("When that health endpoint returns 'Failure'", func() {
			c.Healthcheck = healthFailure
			Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
				c.Checker(ctx, checkState)
				So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
				So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
				So(checkState.StatusCode(), ShouldEqual, 0)
			})
		})
	})

	Convey("Given a CheckMongoClient with collections", t, func() {
		collections := map[health.Database][]health.Collection{"db": {"col1", "col2"}}
		c := &health.CheckMongoClient{
			Client: *health.NewClientWithCollections(nil, collections),
		}
		// CheckState for test validation
		checkState := healthcheck.NewCheckState(health.ServiceName)

		Convey("When that health endpoint returns 'Success'", func() {
			c.Healthcheck = healthSuccess

			Convey("And the collections exist", func() {
				c.CheckCollections = func(context.Context) error {
					return nil
				}
				Convey("Then Checker updates the CheckState to an OK status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusOK)
					So(checkState.Message(), ShouldEqual, "mongodb is OK and all expected collections exist")
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})

			Convey("And the collections do not exist", func() {
				c.CheckCollections = func(context.Context) error {
					return errCollectionsDoNotExist
				}
				Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
					So(checkState.Message(), ShouldEqual, errCollectionsDoNotExist.Error())
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})
		})

		Convey("When that health endpoint returns 'Failure'", func() {
			c.Healthcheck = healthFailure
			Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
				c.Checker(ctx, checkState)
				So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
				So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
				So(checkState.StatusCode(), ShouldEqual, 0)
			})
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
