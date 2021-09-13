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

const (
	testResource         = "image"
	testResourceID       = "1234"
	testResourceName     = "image-1234"
	testLockID           = "image-1234-123456789"
	testUniqueCallerName = "uniqueCallerName"
)

var (
	ctx                = context.Background()
	testGenerateTimeID = func() int {
		return 123456789
	}
)

func TestLock(t *testing.T) {

	// consistent time ID for testing
	dplock.GenerateTimeID = testGenerateTimeID

	Convey("Given a lock with a client that can successfully lock", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := dplock.Lock{
			Resource: testResource,
			Client:   clientMock,
			Cfg: dplock.Config{
				TTL: dplock.DefaultTTL,
			},
		}

		Convey("Calling Lock performs a lock using the underlying client with the expected resource, id and TTL", func() {
			lockID, err := l.Lock(testResourceID, testUniqueCallerName)
			So(err, ShouldBeNil)
			So(lockID, ShouldEqual, testLockID)
			So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, testResourceName)
			So(clientMock.XLockCalls()[0].LockID, ShouldEqual, testLockID)
			So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{
				Owner: testUniqueCallerName,
				TTL:   dplock.DefaultTTL,
			})
		})
	})

	Convey("Given a lock with a client that is already locked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := dplock.Lock{
			Resource: testResource,
			Client:   clientMock,
		}

		Convey("Calling Lock fails with the same error", func() {
			_, err := l.Lock(testResourceID, testUniqueCallerName)
			So(err, ShouldResemble, lock.ErrAlreadyLocked)
		})
	})
}

func TestAcquire(t *testing.T) {

	// consistent time ID and low acquire period for testing
	dplock.GenerateTimeID = testGenerateTimeID

	// aux func to get a testing Lock for acquire with short times and the provided mock client
	testLockWithMock := func(clientMock *mock.ClientMock) dplock.Lock {
		cfg := dplock.Config{
			TTL:                           dplock.DefaultTTL,
			AcquireMinPeriodMillis:        1,
			AcquireMaxPeriodMillis:        2,
			AcquireRetryTimeout:           3 * time.Millisecond,
			MaxCount:                      dplock.DefaultMaxCount,
			TimeThresholdSinceLastRelease: dplock.DefaultTimeThresholdSinceLastRelease,
			UsageSleep:                    dplock.DefaultUsageSleep,
		}
		return dplock.Lock{
			Resource:      testResource,
			Client:        clientMock,
			CloserChannel: make(chan struct{}),
			Cfg:           cfg,
			Usages:        dplock.NewUsages(&cfg),
		}
	}

	Convey("Given a lock with a client that can successfully lock", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("When Acquire is called", func() {
			lockID, err := l.Acquire(ctx, testResourceID, testUniqueCallerName)

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the underlying client locks the expected resource id and TTL and owner", func() {
				So(lockID, ShouldEqual, testLockID)
				So(len(clientMock.XLockCalls()), ShouldEqual, 1)
				So(clientMock.XLockCalls()[0].ResourceName, ShouldEqual, testResourceName)
				So(clientMock.XLockCalls()[0].LockID, ShouldEqual, testLockID)
				So(clientMock.XLockCalls()[0].Ld, ShouldResemble, lock.LockDetails{
					Owner: testUniqueCallerName,
					TTL:   l.Cfg.TTL,
				})
			})

			Convey("Then the successful acquire is accounted for in the Usages struct", func() {
				So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
					testResourceName: {
						testUniqueCallerName: {
							Count: 1,
						},
					},
				})
			})
		})

		Convey("And that MaxCount has been reached in corresponding theUsage struct with a recent last release time", func() {
			slept := []time.Duration{}
			dplock.Sleep = func(d time.Duration) {
				slept = append(slept, d)
			}

			t0 := getUnexpiredTime()
			l.Usages = dplock.NewUsages(cfg)
			l.Usages.UsagesMap = map[string]map[string]*dplock.Usage{
				testResourceName: {
					testUniqueCallerName: {
						Count:    dplock.DefaultMaxCount,
						Released: t0,
					},
				},
			}

			Convey("When Acquire is called", func() {
				_, err := l.Acquire(ctx, testResourceID, testUniqueCallerName)

				Convey("Then no error is returned", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the underlying client is called", func() {
					So(len(clientMock.XLockCalls()), ShouldEqual, 1)
				})

				Convey("Then we sleep for the expected time period", func() {
					So(slept, ShouldHaveLength, 1)
					So(slept[0], ShouldEqual, dplock.DefaultUsageSleep)
				})

				Convey("Then the Usages struct count is reset to 0, then set to 1, and the Released time is not modified", func() {
					So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
						testResourceName: {
							testUniqueCallerName: {
								Count:    1,
								Released: t0,
							},
						},
					})
				})
			})
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

		Convey("When Acquire is called", func() {
			_, err := l.Acquire(ctx, testResourceID, testUniqueCallerName)

			Convey("Then no error is returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the lock is acquired in the second call of the underlying client", func() {
				So(len(clientMock.XLockCalls()), ShouldEqual, 2)
			})

			Convey("Then the acquire is not accounted in the Usages struct", func() {
				So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{})
			})
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

		Convey("When Acquire is called", func() {
			_, err := l.Acquire(ctx, testResourceID, testUniqueCallerName)

			Convey("Then the expected error is returned", func() {
				So(err, ShouldResemble, errLock)
			})

			Convey("Then the acquire is not retried", func() {
				So(len(clientMock.XLockCalls()), ShouldEqual, 1)
			})

			Convey("Then the acquire attempt is not accounted in the Usages struct", func() {
				So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{})
			})
		})
	})

	Convey("Given a lock with a client that always fails with ErrAlreadyLocked", t, func() {
		clientMock := &mock.ClientMock{
			XLockFunc: func(resourceName string, lockID string, ld lock.LockDetails) error {
				return lock.ErrAlreadyLocked
			},
		}
		l := testLockWithMock(clientMock)

		Convey("When Acquire is called", func() {
			_, err := l.Acquire(ctx, testResourceID, testUniqueCallerName)

			Convey("Then 'ErrAcquireTimeout' error is returned", func() {
				So(err, ShouldResemble, dplock.ErrAcquireTimeout)
			})

			Convey("Then acquire is retried multiple times", func() {
				So(len(clientMock.XLockCalls()), ShouldBeGreaterThan, 1) // due to random sleeps, we can't know the exact number of attempts
			})

			Convey("Then the acquire attempts are not accounted in the Usages struct", func() {
				So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{})
			})
		})

		Convey("Then closing the closer channel whilst acquire is trying to acquire the lock, results in the operation being aborted", func() {
			// High period value to prevent race conditions between channel and 'timeout'
			l.Cfg.AcquireMinPeriodMillis = 30000
			l.Cfg.AcquireMaxPeriodMillis = 30001
			var err error
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err = l.Acquire(ctx, testResourceID, testUniqueCallerName)
			}()
			close(l.CloserChannel)
			wg.Wait()
			So(err, ShouldResemble, dplock.ErrMongoDbClosing)
		})
	})
}

func TestUnlock(t *testing.T) {

	status := []lock.LockStatus{
		{
			Owner:    testUniqueCallerName,
			Resource: testResourceName,
		},
	}

	// aux func to get a testing Lock for acquire with short times and the provided mock client
	testLockWithMock := func(clientMock *mock.ClientMock) dplock.Lock {
		cfg := dplock.Config{
			TTL:                           dplock.DefaultTTL,
			UnlockMinPeriodMillis:         1,
			UnlockMaxPeriodMillis:         2,
			UnlockRetryTimeout:            3 * time.Millisecond,
			MaxCount:                      dplock.DefaultMaxCount,
			TimeThresholdSinceLastRelease: dplock.DefaultTimeThresholdSinceLastRelease,
			UsageSleep:                    dplock.DefaultUsageSleep,
		}
		u := dplock.NewUsages(&cfg)
		u.UsagesMap = map[string]map[string]*dplock.Usage{
			testResourceName: {
				testUniqueCallerName: {},
			},
		}
		return dplock.Lock{
			Resource:      testResource,
			Client:        clientMock,
			CloserChannel: make(chan struct{}),
			Cfg:           cfg,
			Usages:        u,
		}
	}

	Convey("Given a lock with a client that can successfully unlock and a valid Usages map", t, func() {
		clientMock := &mock.ClientMock{
			UnlockFunc: func(lockID string) ([]lock.LockStatus, error) {
				return status, nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("When Unlock is called", func() {
			t0 := time.Now()
			l.Unlock(testLockID)

			Convey("Then the underlying client is unlocked with the provided lockID", func() {
				So(len(clientMock.UnlockCalls()), ShouldEqual, 1)
				So(clientMock.UnlockCalls()[0].LockID, ShouldEqual, testLockID)
			})

			Convey("Then the Usages struct Release time is updated", func() {
				So(l.Usages.UsagesMap[testResourceName][testUniqueCallerName].Released, ShouldHappenOnOrBetween, t0, time.Now())
			})
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
				return status, nil
			},
		}
		l := testLockWithMock(clientMock)

		Convey("When Unlock is called", func() {
			t0 := time.Now()
			l.Unlock(testLockID)

			Convey("Then the underlying client is unlocked in the second attempt", func() {
				So(len(clientMock.UnlockCalls()), ShouldEqual, 2)
			})

			Convey("Then the Usages struct Release time is updated", func() {
				So(l.Usages.UsagesMap[testResourceName][testUniqueCallerName].Released, ShouldHappenOnOrBetween, t0, time.Now())
			})
		})
	})

	Convey("Given a lock with a client that always fails to unlock", t, func() {
		clientMock := &mock.ClientMock{
			UnlockFunc: func(lockID string) ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, errors.New("generic unlock error")
			},
		}
		l := testLockWithMock(clientMock)

		Convey("When Unlock is called", func() {
			l.Unlock(testLockID)

			Convey("Then the underlying client tries to unlock multiple times", func() {
				So(len(clientMock.UnlockCalls()), ShouldBeGreaterThan, 1) // due to random sleeps, we can't know the exact number of attempts
			})

			Convey("Then the Usages struct Release time is not updated", func() {
				So(l.Usages.UsagesMap, ShouldResemble, map[string]map[string]*dplock.Usage{
					testResourceName: {
						testUniqueCallerName: {},
					},
				})
			})
		})

		Convey("Then closing the closer channel whilst unlock is trying to unlock the lock, results in the operation being aborted and not retrying it", func() {
			// High period value to prevent race conditions between channel and 'timeout'
			l.Cfg.UnlockMinPeriodMillis = 30000
			l.Cfg.UnlockMaxPeriodMillis = 30001
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.Unlock(testLockID)
			}()
			close(l.CloserChannel)
			wg.Wait() // Make sure the unlock go-routine is done before checking that it only attempted the unlock once
			So(len(clientMock.UnlockCalls()), ShouldEqual, 1)
		})
	})
}

func TestInit(t *testing.T) {
	Convey("Given a Client and a Purger mocks", t, func() {
		clientMock := &mock.ClientMock{}
		purgerMock := &mock.PurgerMock{
			PurgeFunc: func() ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := dplock.Lock{Resource: testResource}

		Convey("Then init the client with nil config results in the default config being generated", func() {
			err := l.Init(ctx, clientMock, purgerMock, nil)
			So(err, ShouldBeNil)
			So(l.Cfg, ShouldResemble, dplock.GetConfig(nil))
		})

		Convey("Then init the client with an invalid config override results in the config validation error being returned", func() {
			var max uint = 10
			var min uint = 100
			err := l.Init(ctx, clientMock, purgerMock, &dplock.ConfigOverride{
				AcquireMaxPeriodMillis: &max,
				AcquireMinPeriodMillis: &min,
			})
			So(err, ShouldResemble, errors.New("acquire max period must be greater than acquire min period"))
		})
	})
}

func TestClose(t *testing.T) {
	Convey("Given a lock initialised with Client and Purger mocks", t, func() {
		clientMock := &mock.ClientMock{}
		purgerMock := &mock.PurgerMock{
			PurgeFunc: func() ([]lock.LockStatus, error) {
				return []lock.LockStatus{}, nil
			},
		}
		l := dplock.Lock{Resource: testResource}
		l.Init(ctx, clientMock, purgerMock, nil)

		Convey("Then executing Close result in the closer channel being closed, and the purger go-routine ends", func() {
			l.Close(ctx)
			l.WaitGroup.Wait()
			So(len(purgerMock.PurgeCalls()), ShouldEqual, 1)
		})
	})
}
