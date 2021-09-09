package dplock_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ONSdigital/dp-mongodb/dplock"
	"github.com/ONSdigital/dp-mongodb/dplock/mock"
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
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
			Config: dplock.Config{
				TTL: dplock.DefaultTTL,
			},
		}

		Convey("Calling Lock performs a lock using the underlying client with the expected resource, id and TTL", func() {
			lockID, err := l.Lock("myID")
			So(err, ShouldBeNil)
			So(lockID, ShouldEqual, "image-myID-123456789")
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, "image-myID")
			So(clientMock.XLockCalls()[0].LockID, ShouldEqual, "image-myID-123456789")
			So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{TTL: dplock.DefaultTTL})
		})
	})

	Convey("Given a lock with a client that is already locked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := dplock.Lock{
			Resource: "image",
			Client:   clientMock,
		}

		Convey("Calling Lock fails with the same error", func() {
			_, err := l.Lock("myID")
			So(err, ShouldResemble, lock.ErrAlreadyLocked)
		})
	})
}

func TestAcquire(t *testing.T) {

	// consistent time ID and low acquire period for testing
	dplock.GenerateTimeID = func() int {
		return 123456789
	}

	// aux func to get a testing Lock for acquire with short times and the provided mock client
	testLockWithMock := func(clientMock *mock.ClientMock) dplock.Lock {
		return dplock.Lock{
			Resource:      "image",
			Client:        clientMock,
			CloserChannel: make(chan struct{}),
			Config: dplock.Config{
				TTL:                    dplock.DefaultTTL,
				AcquireMinPeriodMillis: 1,
				AcquireMaxPeriodMillis: 2,
				AcquireRetryTimeout:    3 * time.Millisecond,
			},
		}
	}

	Convey("Given a lock with a client that can successfully lock", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Acquire performs a lock using the underlying client with the expected resource, id and TTL", func() {
			lockID, err := l.Acquire(ctx, "myID")
			So(err, ShouldBeNil)
			So(lockID, ShouldEqual, "image-myID-123456789")
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, "image-myID")
			So(clientMock.XLockCalls()[0].LockID, ShouldEqual, "image-myID-123456789")
			So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{TTL: l.Config.TTL})
		})
	})

	Convey("Given a lock with a client that fails to lock with ErrAlreadyLocked, only on the first iteration", t, func() {
		i := 0
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				i++
				if i == 1 {
					return lock.ErrAlreadyLocked
				}
				return nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Acquire manages to acquire the lock using the underlying client in the second iteration", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldBeNil)
			So(len(clientMock.XLockCalls()), ShouldEqual, 2)
		})
	})

	Convey("Given a lock with a client that fails with a generic error", t, func() {
		errLock := errors.New("XLock generic error")
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return errLock
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Acquire, fails to acquire locking with the same error, without retrying", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldResemble, errLock)
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
		})
	})

	Convey("Given a lock with a client that always fails with ErrAlreadyLocked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Then after retrying 'AcquireMaxRetries' times, acquire fails with the expected error", func() {
			_, err := l.Acquire(ctx, "myID")
			So(err, ShouldResemble, dplock.ErrAcquireTimeout)
			So(len(clientMock.XLockCalls()), ShouldBeGreaterThan, 1)
		})

		Convey("Then closing the closer channel whilst acquire is trying to acquire the lock, results in the operation being aborted", func() {
			// High period value to prevent race conditions between channel and 'timeout'
			l.Config.AcquireMinPeriodMillis = 30000
			l.Config.AcquireMaxPeriodMillis = 30001
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

	// aux func to get a testing Lock for acquire with short times and the provided mock client
	testLockWithMock := func(clientMock *mock.ClientMock) dplock.Lock {
		return dplock.Lock{
			Resource:      "image",
			Client:        clientMock,
			CloserChannel: make(chan struct{}),
			Config: dplock.Config{
				TTL:                   dplock.DefaultTTL,
				UnlockMinPeriodMillis: 1,
				UnlockMaxPeriodMillis: 2,
				UnlockRetryTimeout:    3 * time.Millisecond,
			},
		}
	}

	Convey("Given a lock with a client that can successfully unlock", t, func() {
		clientMock := &mock.ClientMock{
			UnlockFunc: func(lockID string) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Unlock performs an unlock using the underlying client with the provided lock id", func() {
			l.Unlock("lockID")
			So(len(clientMock.UnlockCalls()), ShouldEqual, 1)
			So(clientMock.UnlockCalls()[0].LockID, ShouldEqual, "lockID")
		})
	})

	Convey("Given a lock with a client that fails to unlock only on the first iteration", t, func() {
		i := 0
		clientMock := &mock.ClientMock{
			UnlockFunc: func(lockID string) ([]lock.LockStatus, error) {
				i++
				if i == 1 {
					return []lock.LockStatus{}, errors.New("generic unlock error")
				}
				return []lock.LockStatus{}, nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Unlock manages to acquire the lock using the underlying client in the second iteration", func() {
			l.Unlock("lockID")
			So(len(clientMock.UnlockCalls()), ShouldEqual, 2)
		})
	})

	Convey("Given a lock with a client that always fails to unlock", t, func() {
		clientMock := &mock.ClientMock{
			UnlockFunc: func(lockID string) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, errors.New("generic unlock error")
			},
		}
		l := testLockWithMock(clientMock)

		Convey("Calling Unlock retries to unlock multiple times until timeout is reached", func() {
			l.Unlock("lockID")
			So(len(clientMock.UnlockCalls()), ShouldBeGreaterThan, 1)
		})

		Convey("Then closing the closer channel whilst unlock is trying to unlock the lock, results in the operation being aborted and not retrying it", func() {
			// High period value to prevent race conditions between channel and 'timeout'
			l.Config.UnlockMinPeriodMillis = 30000
			l.Config.UnlockMaxPeriodMillis = 30001
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.Unlock("lockID")
			}()
			close(l.CloserChannel)
			wg.Wait() // Make sure the unlock go-routine is done before checking that it only attempted the unlock once
			So(len(clientMock.UnlockCalls()), ShouldEqual, 1)
		})
	})
}

func TestLifecycleAndPurger(t *testing.T) {
	Convey("Given a lock initialised with Client and Purger mocks", t, func() {
		clientMock := &mock.ClientMock{}
		purgerMock := &mock.PurgerMock{
			PurgeFunc: func() ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := dplock.Lock{Resource: "image"}
		l.Init(ctx, clientMock, purgerMock, nil)
		// TODO test config override

		Convey("Then executing Close result in the closer channel being closed, and the purger go-routine ends", func() {
			l.Close(ctx)
			l.WaitGroup.Wait()
			So(len(purgerMock.PurgeCalls()), ShouldEqual, 1)
		})
	})
}
