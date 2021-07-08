package dplock

import (
	"context"
	"errors"
	"fmt"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v2/mongodb"
	"sync"
	"time"

	"github.com/ONSdigital/log.go/log"
	lock "github.com/square/mongo-lock"
)

// TTL is the 'time to live' for a lock in number of seconds
const TTL = 30

// PurgerPeriod is the time period between expired lock purges
const PurgerPeriod = 5 * time.Minute

// AcquirePeriod is the time period between acquire lock retries
var AcquirePeriod = 250 * time.Millisecond

// AcquireMaxRetries is the maximum number of locking retries by the Acquire lock, discounting the first attempt
var AcquireMaxRetries = 10

// ErrMongoDbClosing is an error returned because MongoDB is being closed
var ErrMongoDbClosing = errors.New("mongo db is being closed")

// ErrAcquireMaxRetries is an error returned when acquire fails
// after retrying to lock a resource 'AcquireMaxRetries' times
var ErrAcquireMaxRetries = errors.New("cannot acquire lock, maximum number of retries has been reached")

//go:generate moq -out mock/client.go -pkg mock . Client
//go:generate moq -out mock/purger.go -pkg mock . Purger

//Client defines the lock Client methods from mongo-lock
type Client interface {
	XLock(ctx context.Context, resourceName, lockID string, ld lock.LockDetails) error
	Unlock(ctx context.Context, lockID string) ([]lock.LockStatus, error)
}

// Purger defines the lock Purger methods from mongo-lock
type Purger interface {
	Purge(ctx context.Context) ([]lock.LockStatus, error)
}

// Lock is a MongoDB lock for a resource
type Lock struct {
	Client        Client
	CloserChannel chan struct{}
	Purger        Purger
	WaitGroup     *sync.WaitGroup
	Resource      string
}

// GenerateTimeID returns the current timestamp in nanoseconds
var GenerateTimeID = func() int {
	return time.Now().Nanosecond()
}

// New creates a new mongoDB lock for the provided session, db, collection and resource
func New(ctx context.Context, mongoConnection *mongoDriver.MongoConnection, resource string) *Lock {

	lockClient := lock.NewClient(mongoConnection.GetMongoCollection())
	lockClient.CreateIndexes(ctx)
	lockPurger := lock.NewPurger(lockClient)
	lck := &Lock{
		Resource: resource,
	}
	lck.Init(ctx, lockClient, lockPurger)

	return lck
}

// Init initialises a lock with the provided client and purger, and starts the purger loop
func (l *Lock) Init(ctx context.Context, lockClient Client, lockPurger Purger) {
	l.Client = lockClient
	l.Purger = lockPurger
	l.CloserChannel = make(chan struct{})
	l.WaitGroup = &sync.WaitGroup{}
	l.startPurgerLoop(ctx)
}

// startPurgerLoop creates a go-routine which periodically performs a lock Purge, which removes expired locks
// if closerChannel is closed, this go-routine finishes its execution, releasing its WaitGroup delta.
func (l *Lock) startPurgerLoop(ctx context.Context) {
	l.WaitGroup.Add(1)
	go func() {
		defer l.WaitGroup.Done()
		for {
			l.Purger.Purge(ctx)
			select {
			case <-l.CloserChannel:
				log.Event(ctx, "closing mongo db lock purger go-routine", log.INFO)
				return
			case <-time.After(PurgerPeriod):
				log.Event(ctx, "purging expired mongoDB locks", log.INFO)
			}
		}
	}()
}

// Lock acquires an exclusive mongoDB lock with the provided id, with the default TTL value.
// If the resource is already locked, an error will be returned.
func (l *Lock) Lock(ctx context.Context, resourceID string) (lockID string, err error) {
	lockID = fmt.Sprintf("%s-%s-%d", l.Resource, resourceID, GenerateTimeID())
	return lockID, l.Client.XLock(ctx,
		fmt.Sprintf("%s-%s", l.Resource, resourceID),
		lockID,
		lock.LockDetails{TTL: TTL},
	)
}

// Acquire tries to lock the provided id.
// If the resource is already locked, this function will block until the existing lock is released,
// at which point we acquire the lock and return.
func (l *Lock) Acquire(ctx context.Context, id string) (lockID string, err error) {
	retries := 0
	for {
		lockID, err = l.Lock(ctx, id)
		if err != lock.ErrAlreadyLocked {
			return lockID, err
		}
		if retries >= AcquireMaxRetries {
			return "", ErrAcquireMaxRetries
		}
		retries++
		select {
		case <-time.After(AcquirePeriod):
			continue
		case <-l.CloserChannel:
			log.Event(ctx, "stop acquiring lock. Mongo db is being closed", log.INFO)
			return "", ErrMongoDbClosing
		}
	}
}

// Unlock releases an exclusive mongoDB lock for the provided id (if it exists)
func (l *Lock) Unlock(ctx context.Context, lockID string) error {
	_, err := l.Client.Unlock(ctx, lockID)
	return err
}

// Close closes the closer channel, and waits for the WaitGroup to finish.
func (l *Lock) Close(ctx context.Context) {
	close(l.CloserChannel)
	l.WaitGroup.Wait()
}
