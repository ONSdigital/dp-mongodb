package dplock_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ONSdigital/dp-mongodb/v2/dplock"
	mock "github.com/ONSdigital/dp-mongodb/v2/dplock/mock"
	. "github.com/smartystreets/goconvey/convey"
	lock "github.com/square/mongo-lock"
)

var ctx = context.Background()

func TestLock(t *testing.T) {

	// consistent time ID for testing
	dplock.GenerateTimeID = func() int {
		return 123456789
	}

	Convey("Given a lock with a client that can successfully lock", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Lock performs a lock using the underlying client with the expected resource, id and TTL", func() {
			lockID, err := l.Lock(ctx, "myID")
			So(err, ShouldBeNil)
			So(lockID, ShouldEqual, "image-myID-123456789")
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, "image-myID")
			So(clientMock.XLockCalls()[0].LockID, ShouldEqual, "image-myID-123456789")
			So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{TTL: dplock.TTL})
		})
	})

	Convey("Given a lock with a client that is already locked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Lock fails with the same error", func() {
			_, err := l.Lock(ctx, "myID")
			So(err, ShouldResemble, lock.ErrAlreadyLocked)
		})
	})
}

func TestAcquire(t *testing.T) {

	// consistent time ID and low acquire period for testing
	dplock.GenerateTimeID = func() int {
		return 123456789
	}
	dplock.AcquirePeriod = 1 * time.Nanosecond
	dplock.AcquireMaxRetries = 5

	Convey("Given a lock with a client that can successfully lock", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Acquire performs a lock using the underlying client with the expected resource, id and TTL", func() {
			lockID, err := l.Acquire(ctx, "myID")
			So(err, ShouldBeNil)
			So(lockID, ShouldEqual, "image-myID-123456789")
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, "image-myID")
			So(clientMock.XLockCalls()[0].LockID, ShouldEqual, "image-myID-123456789")
			So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{TTL: dplock.TTL})
		})
	})

	Convey("Given a lock with a client that fails to lock with ErrAlreadyLocked, only on the first iteration", t, func() {
		i := 0
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				i++
				if i == 1 {
					return lock.ErrAlreadyLocked
				}
				return nil
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Acquire manages to acquire the lock using the underlying client in the second iteration", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldBeNil)
			So(len(clientMock.XLockCalls()), ShouldEqual, 2)
		})
	})

	Convey("Given a lock with a client that fails with a generic error", t, func() {
		errLock := errors.New("XLock generic error")
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				return errLock
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Acquire, fails to acquire locking with the same error, without retrying", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldResemble, errLock)
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
		})
	})

	Convey("Given a lock with a client that always fails with ErrAlreadyLocked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(ctx context.Context, resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := dplock.Lock{
			Resource:      "image",
			Client:        clientMock,
			CloserChannel: make(chan struct{}),
		}

		Convey("Then after retrying 'AcquireMaxRetries' times, acquire fails with the expected error", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldResemble, dplock.ErrAcquireMaxRetries)
			So(len(clientMock.XLockCalls()), ShouldEqual, 6)
		})

		Convey("Then closing the closer channel whilst acquire is trying to acquire the lock, results in the operation being aborted", func() {
			// High period value to prevent race conditions between channel and 'timeout'
			dplock.AcquirePeriod = 30 * time.Second
			var err error
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err = l.Acquire(ctx, "myID")
			}()
			close(l.CloserChannel)
			wg.Wait()
			So(err, ShouldResemble, dplock.ErrMongoDbClosing)
		})
	})
}

func TestUnlock(t *testing.T) {
	Convey("Given a lock with a client that can successfully unlock", t, func() {
		clientMock := &mock.ClientMock{
			UnlockFunc: func(ctx context.Context, lockID string) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Unlock performs an unlock using the underlying client with the provided lock id", func() {
			err := l.Unlock(ctx, "lockID")
			So(err, ShouldBeNil)
			So(len(clientMock.UnlockCalls()), ShouldEqual, 1)
			So(clientMock.UnlockCalls()[0].LockID, ShouldEqual, "lockID")
		})
	})

	Convey("Given a lock with a client that fails to unlock", t, func() {
		errUnlock := errors.New("generic unlock error")
		clientMock := &mock.ClientMock{
			UnlockFunc: func(ctx context.Context, lockID string) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, errUnlock
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Unlock fails with the same error", func() {
			err := l.Unlock(ctx, "myID")
			So(err, ShouldResemble, errUnlock)
		})
	})
}

func TestLifecycleAndPurger(t *testing.T) {
	Convey("Given a lock initialised with Client and Purger mocks", t, func() {
		clientMock := &mock.ClientMock{}
		purgerMock := &mock.PurgerMock{
			PurgeFunc: func(ctx context.Context) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := dplock.Lock{Resource: "image"}
		l.Init(ctx, clientMock, purgerMock)

		Convey("Then executing Close result in the closer channel being closed, and the purger go-routine ends", func() {
			l.Close(ctx)
			l.WaitGroup.Wait()
			So(len(purgerMock.PurgeCalls()), ShouldEqual, 1)
		})
	})
}
