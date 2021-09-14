package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ONSdigital/dp-mongodb/dplock"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/health"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/globalsign/mgo"
)

// MongoURI is the URI to connect to MongoDB
const MongoURI = "localhost:27017"

// Database is the MongoDB database
const Database = "datasets"

// Collections are the MongoDB collections that will be used
const (
	datasetsCollection         = "datasets"
	editionsCollection         = "editions"
	instanceCollection         = "instances"
	instanceLockCollection     = "instances_locks"
	dimensionOptionsCollection = "dimension.options"
)

// Testing parameters
const (
	NumCallers                 = 2                     // Number of concurent go-routines that try to acquire and release the lock
	WorkPerCaller              = 100                   // Number of times each caller needs to acquire and release the lock.
	SleepTime                  = 20 * time.Millisecond // Amount of time that each worker will sleep after successfully acquiring a lock, before releaseing it
	SleepTimeBetweenIterations = 20 * time.Millisecond // Amount of time that each worker will sleep after successfully releasing a lock, before the next iteration
)

// getConfig is the dplock config override to be able to control the locking algorithm
func getConfig() *dplock.ConfigOverride {
	// th := 3 * time.Second
	// sl := 5 * time.Second
	// var maxCount uint = 1000
	// var min uint = 1000
	// var max uint = 1001
	tout := 1000 * time.Millisecond
	return &dplock.ConfigOverride{
		AcquireRetryTimeout: &tout,
		// MaxCount:               &maxCount,
		// AcquireMinPeriodMillis: &min,
		// AcquireMaxPeriodMillis: &max,
		// TimeThresholdSinceLastRelease: &th,
		// UsageSleep:                    &sl,
	}
}

func getMongoDB(ctx context.Context) (*Mongo, error) {
	mongodb := &Mongo{URI: MongoURI}
	if err := mongodb.Init(ctx, getConfig()); err != nil {
		return nil, err
	}
	log.Info(ctx, "listening to mongo db session", log.Data{"URI": mongodb.URI})
	return mongodb, nil
}

func main() {
	log.Namespace = "dp-mongodb-lock-stress-test"
	ctx := context.Background()

	m, err := getMongoDB(ctx)
	if err != nil {
		log.Error(ctx, "failed to initialise dplock", err)
		os.Exit(1)
	}

	doTest(ctx, m)
	log.Info(ctx, "testing has finished", log.Data{"usages": m.lockClient.Usages.UsagesMap})
}

func doTest(ctx context.Context, m *Mongo) {
	wg := &sync.WaitGroup{}

	instanceID := "testInstance"
	t0 := time.Now()

	for i := 0; i < NumCallers; i++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			workDone := 0
			for {
				lockID, err := m.lockClient.Acquire(ctx, instanceID, workerID)
				if err != nil {
					log.Error(ctx, "worker failed to acquire lock", err, log.Data{"worker_id": workerID})
					os.Exit(2)
				}
				log.Info(ctx, "lock has been acquired", log.Data{"worker_id": workerID, "time": time.Since(t0).Milliseconds()})
				time.Sleep(SleepTime)
				m.lockClient.Unlock(lockID)
				workDone++
				if workDone == WorkPerCaller {
					log.Info(ctx, "worker has finished its work", log.Data{"worker_id": workerID})
					return
				}
				time.Sleep(SleepTimeBetweenIterations)
			}
		}(fmt.Sprintf("%d", i))
	}

	wg.Wait()
}

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Session      *mgo.Session
	URI          string
	healthClient *dpMongoHealth.CheckMongoClient
	lockClient   *dplock.Lock
}

// Init creates a new mgo.Session with a strong consistency and a write mode of "majortiy"; and initialises the mongo health client.
func (m *Mongo) Init(ctx context.Context, cfg *dplock.ConfigOverride) (err error) {
	if m.Session != nil {
		return errors.New("session already exists")
	}

	// Create session
	if m.Session, err = mgo.Dial(m.URI); err != nil {
		return err
	}
	m.Session.EnsureSafe(&mgo.Safe{WMode: "majority"})
	m.Session.SetMode(mgo.Strong, true)

	databaseCollectionBuilder := make(map[dpMongoHealth.Database][]dpMongoHealth.Collection)
	databaseCollectionBuilder[(dpMongoHealth.Database)(Database)] = []dpMongoHealth.Collection{(dpMongoHealth.Collection)(datasetsCollection), (dpMongoHealth.Collection)(editionsCollection), (dpMongoHealth.Collection)(instanceCollection), (dpMongoHealth.Collection)(instanceLockCollection), (dpMongoHealth.Collection)(dimensionOptionsCollection)}

	// Create client and healthclient from session
	client := dpMongoHealth.NewClientWithCollections(m.Session, databaseCollectionBuilder)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *client,
		Healthcheck: client.Healthcheck,
	}

	// Create MongoDB lock client with the provided config override
	m.lockClient, err = dplock.New(ctx, m.Session, Database, instanceCollection, cfg)
	return err
}
