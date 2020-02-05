package mongo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ONSdigital/log.go/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSuccessfulCloseMongoSession(t *testing.T) {
	_, err := setupSession()
	if err != nil {
		log.Event(nil, "mongo instance not available, skip close tests", log.Error(err))
		return
	}

	if err = cleanupTestData(session.Copy()); err != nil {
		log.Event(nil, "failed to delete test data", log.Error(err))
	}

	Convey("Safely close mongo session", t, func() {
		if !hasSessionSleep {
			Convey("with no context deadline", func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				err := Close(ctx, session.Copy())

				So(err, ShouldBeNil)
			})
		}

		Convey("within context timeout (deadline)", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := Close(ctx, session.Copy())

			So(err, ShouldBeNil)
		})

		Convey("within context deadline", func() {
			time := time.Now().Local().Add(time.Second * time.Duration(2))
			ctx, cancel := context.WithDeadline(context.Background(), time)
			defer cancel()
			err := Close(ctx, session.Copy())

			So(err, ShouldBeNil)
		})
	})

	if err = setUpTestData(session.Copy()); err != nil {
		log.Event(nil, "failed to insert test data, skipping tests", log.Error(err))
		os.Exit(1)
	}

	Convey("Timed out from safely closing mongo session", t, func() {
		Convey("with no context deadline", func() {
			start = ungraceful{}
			copiedSession := session.Copy()
			go func() {
				_ = slowQueryMongo(copiedSession)
			}()
			// Sleep for half a second for mongo query to begin
			time.Sleep(500 * time.Millisecond)

			ctx := context.WithValue(context.Background(), returnContextKey, early)
			err := Close(ctx, copiedSession)

			So(err, ShouldNotBeNil)
			So(err, ShouldResemble, errors.New("closing mongo timed out"))
			time.Sleep(500 * time.Millisecond)
		})

		Convey("with context deadline", func() {
			copiedSession := session.Copy()
			go func() {
				_ = slowQueryMongo(copiedSession)
			}()
			// Sleep for half a second for mongo query to begin
			time.Sleep(500 * time.Millisecond)

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			err := Close(ctx, copiedSession)

			So(err, ShouldNotBeNil)
			So(err, ShouldResemble, context.DeadlineExceeded)
		})
	})

	if err = cleanupTestData(session.Copy()); err != nil {
		log.Event(nil, "failed to delete test data", log.Error(err))
	}
}
