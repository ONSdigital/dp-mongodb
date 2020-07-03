package dplock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ONSdigital/log.go/log"
	"github.com/globalsign/mgo"
	lock "github.com/square/mongo-lock"
)

// TTL is the 'time to live' for a lock in number of seconds
const TTL = 30

// PurgerPeriod is the time period between expired lock purges
var PurgerPeriod = 10 * time.Second

// AcquirePeriod is the time period between acquire lock retries
var AcquirePeriod = 100 * time.Millisecond

// ErrMongoDbClosing is an error returned because MongoDB is being closed
var ErrMongoDbClosing = errors.New("mongo db is being closed")

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
}

// GenerateTimeID returns the current timestamp in nanoseconds
var GenerateTimeID = func() int {
	return time.Now().Nanosecond()
}

// New creates a new mongoDB lock for the provided session, db, collection and resource
func New(ctx context.Context, session *mgo.Session, db, resource string) *Lock {
	lockClient := lock.NewClient(session, db, fmt.Sprintf("%s_locks", resource))
	lockClient.CreateIndexes()
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
			l.Purger.Purge()
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
func (l *Lock) Lock(id string) error {
	return l.Client.XLock(
		fmt.Sprintf("%s-%s", l.Resource, id),
		fmt.Sprintf("%s-%s-%d", l.Resource, id, GenerateTimeID()),
		lock.LockDetails{TTL: TTL},
	)
}

// Acquire tries to lock the provided id.
// If the resource is already locked, this function will block until the existing lock is released,
// at which point we acquire the lock and return.
func (l *Lock) Acquire(ctx context.Context, id string) error {
	for {
		if err := l.Lock(id); err != lock.ErrAlreadyLocked {
			return err
		}
		select {
		case <-time.After(AcquirePeriod):
			continue
		case <-l.CloserChannel:
			log.Event(ctx, "stop acquiring lock. Mongo db is being closed", log.INFO)
			return ErrMongoDbClosing
		}
	}
}

// Unlock releases an exclusive mongoDB lock for the provided id (if it exists)
func (l *Lock) Unlock(id string) error {
	_, err := l.Client.Unlock(fmt.Sprintf("%s-%s", l.Resource, id))
	return err
}

// Close closes the closer channel, and waits for the WaitGroup to finish.
func (l *Lock) Close(ctx context.Context) {
	close(l.CloserChannel)
	l.WaitGroup.Wait()
}
