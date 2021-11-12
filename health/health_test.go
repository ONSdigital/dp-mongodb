package health

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	errUnableToConnect = errors.New("failed to connect")
)

func TestClient_GetOutput(t *testing.T) {

	ctx := context.Background()

	Convey("Given a CheckMongoClient", t, func() {
		// CheckState for test validation
		checkState := healthcheck.NewCheckState("test-mongodb")
		Convey("Without collections", func() {
			c := NewClient(nil)

			Convey("When the health endpoint is successful", func() {
				c.healthcheck = healthSuccess
				Convey("Then Checker updates the CheckState to an OK status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusOK)
					So(checkState.Message(), ShouldEqual, "mongodb is OK")
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})

			Convey("When the health endpoint returns an error", func() {
				c.healthcheck = healthFailure
				Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
					So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})
		})

		Convey("With an empty collections map", func() {
			c := NewClientWithCollections(nil, nil)

			Convey("When the health endpoint is successful", func() {
				c.healthcheck = healthSuccess
				Convey("Then Checker updates the CheckState to an OK status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusOK)
					So(checkState.Message(), ShouldEqual, "mongodb is OK")
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})

			Convey("When the health endpoint returns an error", func() {
				c.healthcheck = healthFailure
				Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
					So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})
		})

		Convey("With collections", func() {
			collections := map[Database][]Collection{"db": {"col1", "col2"}}
			c := NewClientWithCollections(nil, collections)

			Convey("When the health endpoint is successful", func() {
				c.healthcheck = healthSuccess

				Convey("And the collections exist", func() {
					c.checkCollections = func(context.Context) error {
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
					errCollectionsDoNotExist := errors.New("can't find collection")
					c.checkCollections = func(context.Context) error {
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

			Convey("When the health endpoint returns an error", func() {
				c.healthcheck = healthFailure
				Convey("Then Checker updates the CheckState to a CRITICAL status", func() {
					c.Checker(ctx, checkState)
					So(checkState.Status(), ShouldEqual, healthcheck.StatusCritical)
					So(checkState.Message(), ShouldEqual, errUnableToConnect.Error())
					So(checkState.StatusCode(), ShouldEqual, 0)
				})
			})
		})
	})
}

var (
	healthSuccess = func(context.Context) error {
		return nil
	}

	healthFailure = func(context.Context) error {
		return errUnableToConnect
	}
)
