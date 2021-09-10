package dplock

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ONSdigital/log.go/v2/log"
	"github.com/globalsign/mgo"
	lock "github.com/square/mongo-lock"
)

// AcquireRetryLogThreshold is the period of time after which an acquire retry will be logged
const AcquireRetryLogThreshold = 500 * time.Millisecond

// UnlockRetryLogThreshold is the period of time after which an unlock retry will be logged
const UnlockRetryLogThreshold = 100 * time.Millisecond

// ErrMongoDbClosing is an error returned because MongoDB is being closed
var ErrMongoDbClosing = errors.New("mongo db is being closed")

// ErrAcquireTimeout is an error returned when acquire fails
// after retrying to lock a resource for a period of time greater or equal than 'AcquireRetryTimeout'
var ErrAcquireTimeout = errors.New("cannot acquire lock, acquire retry timeout has expired")

// ErrUnlockTimeout is an error logged when unlock fails
// after retrying to unlock a resource for a period of time greater or equal than 'UnlockRetryTimeout'
var ErrUnlockTimeout = errors.New("cannot unlock, unlock retry timeout has expired")

//go:generate moq -out mock/client.go -pkg mock . Client
//go:generate moq -out mock/purger.go -pkg mock . Purger

//Client defines the lock Client methods from mongo-lock
type Client interface {
	XLock(resourceName, lockID string, ld lock.LockDetails) error
	Unlock(lockID string) ([]lock.LockStatus, error)
}

// Purger defines the lock Purger methods from mongo-lock
type Purger interface {
	Purge() ([]lock.LockStatus, error)
}

// Lock is a MongoDB lock for a resource
type Lock struct {
	Client        Client
	CloserChannel chan struct{}
	Purger        Purger
	WaitGroup     *sync.WaitGroup
	Resource      string
	Config        Config
	Usages        Usages
}

// GenerateTimeID returns the current timestamp in nanoseconds
var GenerateTimeID = func() int {
	return time.Now().Nanosecond()
}

// New creates a new mongoDB lock for the provided session, db, collection and resource
func New(ctx context.Context, session *mgo.Session, db, resource string, cfg *ConfigOverride) *Lock {
	lockClient := lock.NewClient(session, db, fmt.Sprintf("%s_locks", resource))
	lockClient.CreateIndexes()
	lockPurger := lock.NewPurger(lockClient)
	lck := &Lock{
		Resource: resource,
	}
	lck.Init(ctx, lockClient, lockPurger, cfg)
	return lck
}

// Init initialises a lock with the provided client, purger and config, and starts the purger loop
func (l *Lock) Init(ctx context.Context, lockClient Client, lockPurger Purger, cfg *ConfigOverride) {
	l.Client = lockClient
	l.Purger = lockPurger
	l.CloserChannel = make(chan struct{})
	l.WaitGroup = &sync.WaitGroup{}
	l.Usages = Usages{}
	l.Config = GetConfig(cfg)
	l.startPurgerLoop(ctx)
}

// startPurgerLoop creates a go-routine which periodically performs a lock Purge, which removes expired locks
// if closerChannel is closed, this go-routine finishes its execution, releasing its WaitGroup delta.
func (l *Lock) startPurgerLoop(ctx context.Context) {
	l.WaitGroup.Add(1)
	go func() {
		defer l.WaitGroup.Done()
		for {
			l.Purger.Purge()
			l.Usages.Purge()
			select {
			case <-l.CloserChannel:
				log.Info(ctx, "closing mongo db lock purger go-routine")
				return
			case <-time.After(l.Config.PurgerPeriod):
				log.Info(ctx, "purging expired mongoDB locks")
			}
		}
	}()
}

// Lock acquires an exclusive mongoDB lock with the provided id, with the default TTL value.
// If the resource is already locked, an error will be returned.
func (l *Lock) Lock(resourceID, owner string) (lockID string, err error) {
	lockID = fmt.Sprintf("%s-%s-%d", l.Resource, resourceID, GenerateTimeID())
	return lockID, l.Client.XLock(
		l.GetResourceName(resourceID),
		// fmt.Sprintf("%s-%s", l.Resource, resourceID),
		lockID,
		lock.LockDetails{
			Owner: owner,
			TTL:   l.Config.TTL,
		},
	)
}

// GetResourceName generates a resource name by using the lock Resource and the provided resourceID
func (l *Lock) GetResourceName(resourceID string) string {
	return fmt.Sprintf("%s-%s", l.Resource, resourceID)
}

// Acquire tries to lock the provided resourceID.
// If the resource is already locked, this function will block until the existing lock is released,
// at which point we acquire the lock and return.
func (l *Lock) Acquire(ctx context.Context, resourceID, owner string) (lockID string, err error) {
	retries := 0
	var t0 time.Time

	// logIfNeeded is an aux func to log if a successful Acquire took more than AcquireRetryLogThreshold time (after some retries)
	var logIfNeeded = func() {
		if err == nil && retries > 0 { // t0 is set after the first attempt only
			timeSinceFirstAttempt := time.Since(t0)               // time since the first attempt failed
			if timeSinceFirstAttempt > AcquireRetryLogThreshold { // if the time is greater than a threshold, log it
				log.Warn(ctx, "successfully acquired a lock after retrying for an unusually long period of time", log.Data{
					"resource_id":          resourceID,
					"lock_id":              lockID,
					"acquire_retry_period": timeSinceFirstAttempt,
					"num_retries":          retries,
				})
			}
		}
	}

	// if the same caller has acquired a lock lots of times, we may need to wait to give other callers the opportunity to acquire it.
	l.Usages.WaitIfNeeded(l.GetResourceName(resourceID), owner)

	for {
		// Try to acquire the lock
		lockID, err = l.Lock(resourceID, owner)
		if err != lock.ErrAlreadyLocked {
			logIfNeeded()
			if err == nil && retries == 0 {
				l.Usages.SetCount(l.GetResourceName(resourceID), owner) // obtained it straight away
			}
			return lockID, err // Successful or failed due to some generic error (not ErrAlreadyLocked)
		}

		// get the time if it's the first attempt, otherwise, check if the timeout has expired
		if retries == 0 {
			// Save initial time only after the first attempt has failed due to ErrAlreadyLocked
			// to prevent degrading performance in the vast majority of cases where the first attempt will be successful
			t0 = time.Now()
		} else {
			if time.Since(t0) >= l.Config.AcquireRetryTimeout {
				return "", ErrAcquireTimeout // Acquire timeout has expired, aborting.
			}
		}
		retries++

		select {
		case <-time.After(randomDuration(l.Config.AcquireMinPeriodMillis, l.Config.AcquireMaxPeriodMillis)):
			continue // Retry
		case <-l.CloserChannel:
			log.Info(ctx, "stop acquiring lock. Mongo db is being closed")
			return "", ErrMongoDbClosing // Abort because the app is closing
		}
	}
}

// Unlock releases an exclusive mongoDB lock for the provided id (if it exists)
func (l *Lock) Unlock(lockID string) {
	retries := 0
	var t0 time.Time
	var err error
	ctx := context.Background()

	// logIfNeeded is an aux func to log if a successful Acquire took more than AcquireRetryLogThreshold time (after some retries)
	var logIfNeeded = func() {
		if err == nil && retries > 0 { // t0 is set after the first attempt only
			timeSinceFirstAttempt := time.Since(t0)              // time since the first attempt failed
			if timeSinceFirstAttempt > UnlockRetryLogThreshold { // if the time is greater than a threshold, log it
				log.Warn(ctx, "successfully unlocked after retrying for an unusually long period of time", log.Data{
					"lock_id":              lockID,
					"acquire_retry_period": timeSinceFirstAttempt,
					"num_retries":          retries,
				})
			}
		}
	}

	for {
		// Try to unlock the lock
		status, err := l.Client.Unlock(lockID)
		log.Event(ctx, "========= DEBUG == Unlock ok", log.INFO, log.Data{"status": status})
		if err == nil {
			if len(status) > 0 {
				l.Usages.SetReleased(status[0].Resource, status[0].Owner, time.Now())
				log.Event(ctx, "+++++++++++ DEBUG after SetReleased", log.INFO, log.Data{"usages": l.Usages})
			}
			logIfNeeded()
			return // Successful unlock
		}

		// get the time if it's the first attempt, otherwise, check if the timeout has expired
		if retries == 0 {
			// Save initial time only after the first attempt has failed
			// to prevent degrading performance in the vast majority of cases where the first attempt will be successful
			t0 = time.Now()
		} else {
			if time.Since(t0) >= l.Config.UnlockRetryTimeout {
				log.Error(ctx, "error unlocking", ErrUnlockTimeout)
				return // Unlock timeout has expired, aborting.
			}
		}
		retries++

		select {
		case <-time.After(randomDuration(l.Config.UnlockMinPeriodMillis, l.Config.UnlockMaxPeriodMillis)):
			continue // Retry
		case <-l.CloserChannel:
			log.Info(ctx, "stop unlocking lock. Mongo db is being closed", log.INFO)
			return // Abort because the app is closing
		}
	}
}

// Close closes the closer channel, and waits for the WaitGroup to finish.
func (l *Lock) Close(ctx context.Context) {
	close(l.CloserChannel)
	l.WaitGroup.Wait()
}

// randomDuration will return a random time.Duration between minMillis [ms] and maxMillis [ms]
func randomDuration(minMillis, maxMillis uint) time.Duration {
	rand.Seed(time.Now().Unix())
	periodMillis := rand.Intn(int(maxMillis-minMillis)) + int(minMillis)
	return time.Duration(periodMillis) * time.Millisecond
}
