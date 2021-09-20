package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/ONSdigital/dp-mongodb/v2/dplock"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/v2/health"
	dpMongoDriver "github.com/ONSdigital/dp-mongodb/v2/mongodb"
	"github.com/ONSdigital/log.go/v2/log"
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

// MongoDB constants
const (
	connectTimeoutInSeconds = 5
	queryTimeoutInSeconds   = 15
)

// Global variables needed by tests
var (
	maxInstances         int           = 6
	globalMaxAcquireTime time.Duration = 0
	globalMinAcquireTime time.Duration = time.Hour
	mutex                *sync.RWMutex = &sync.RWMutex{}
	aborting             bool          = false
)

// SetMaxTime updates the global maximum acquire time if the provided value is greater than the current max
// this method is concurrency safe
func SetMinMaxTime(t time.Duration) {
	mutex.Lock()
	defer mutex.Unlock()
	if t > globalMaxAcquireTime {
		globalMaxAcquireTime = t
	}
	if t < globalMinAcquireTime {
		globalMinAcquireTime = t
	}
}

// TestConfig defines the configuration for a particular test
type TestConfig struct {
	NumCallers                 int           // Number of concurrent go-routines that try to acquire a lock
	WorkPerCaller              int           // Number of times each caller needs to acquire and release the lock.
	SleepTime                  time.Duration // Amount of time that each worker will sleep after successfully acquiring a lock, before releaseing it
	SleepTimeBetweenIterations time.Duration // Amount of time that each worker will sleep after successfully releasing a lock, before the next iteration
}

// getLockConfig is the dplock config override to be able to control the locking algorithm
// We can tweak the paramters and validate the tests accordingly
func getLockConfig() *dplock.ConfigOverride {
	var (
		maxCount               uint          = math.MaxUint32           // this will prevent the 'sleep after maxCount successful acquires' to not be triggered
		acquireMinPeriodMillis uint          = 5                        // old AcquirePeriod = 250 * time.Millisecond
		acquireMaxPeriodMillis uint          = 20                       // old AcquirePeriod = 250 * time.Millisecond
		acquireRetryTimeout    time.Duration = 10000 * time.Millisecond // old: 10 retries * 250 ms between retries (effectively 2.5s)
		unlockMinPeriodMillis  uint          = 5
		unlockMaxPeriodMillis  uint          = 6
	)

	return &dplock.ConfigOverride{
		MaxCount:               &maxCount,
		AcquireMinPeriodMillis: &acquireMinPeriodMillis,
		AcquireMaxPeriodMillis: &acquireMaxPeriodMillis,
		AcquireRetryTimeout:    &acquireRetryTimeout,
		UnlockMinPeriodMillis:  &unlockMinPeriodMillis,
		UnlockMaxPeriodMillis:  &unlockMaxPeriodMillis,
	}
}

func getMongoDB(ctx context.Context) (*Mongo, error) {
	mongodb := &Mongo{URI: MongoURI}
	if err := mongodb.Init(ctx, false, false, getLockConfig()); err != nil {
		return nil, err
	}
	log.Info(ctx, "listening to mongo db session", log.Data{"URI": mongodb.URI})
	return mongodb, nil
}

func main() {
	log.Namespace = "dp-mongodb-lock-stress-test"
	ctx := context.Background()

	// Create an array of connections to MongoDB (of size maxInstances)
	m, err := getMongoDB(ctx)
	if err != nil {
		log.Error(ctx, "failed to initialise dplock", err)
		os.Exit(1)
	}

	// Purge any existing lock before starting the test
	m.lockClient.Purger.Purge(ctx)

	// default testCfg to be used as a base config for tests
	testCfg := &TestConfig{
		NumCallers:                 1,
		WorkPerCaller:              100,
		SleepTime:                  20 * time.Millisecond,
		SleepTimeBetweenIterations: 250 * time.Millisecond,
	}

	// 1 callers per instance, 1 instances
	testCfg.NumCallers = 2
	log.Info(ctx, "+++ 1. New Test starting +++ 2 callers per instance / 1 instance", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap})
	globalMaxAcquireTime = 0
	globalMinAcquireTime = time.Hour
	t0 := time.Now()
	runTestInstance(ctx, m, testCfg)
	t1 := time.Since(t0)
	m.lockClient.Purger.Purge(ctx)
	if aborting {
		log.Info(ctx, "=== test 1. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 1. [OK] ===", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 10 callers per instance, 1 instances
	testCfg.NumCallers = 10
	log.Info(ctx, "+++ 2. New Test starting +++ 10 callers per instance / 1 instance", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap})
	globalMaxAcquireTime = 0
	globalMinAcquireTime = time.Hour
	t0 = time.Now()
	runTestInstance(ctx, m, testCfg)
	t1 = time.Since(t0)
	m.lockClient.Purger.Purge(ctx)
	if aborting {
		log.Info(ctx, "=== test 2. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 2. [OK] ===", log.Data{"test_config": testCfg, "usages": m.lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")
}

// runTestInstance runs multiple callers in parallel, each one does a sing lock and release,
// once all have finished, a new batch of callers is created, until all the lock+release have been executed
func runTestInstance(ctx context.Context, m *Mongo, cfg *TestConfig) {
	wg := &sync.WaitGroup{}
	instanceID := "testInstance"

	for j := 0; j < cfg.WorkPerCaller; j++ {
		log.Info(ctx, "starting new batch of workers")
		for i := 0; i < cfg.NumCallers; i++ {
			wg.Add(1)
			go func(workerID string) {
				defer wg.Done()
				logData := log.Data{
					"worker_id": workerID,
				}
				// Check if we need to abort test (due to some other go-routine having failed)
				if aborting {
					log.Info(ctx, "exiting go-routine because the test is being aborted ...", logData)
					return
				}

				// Acquire lock
				t0 := time.Now()
				lockID, err := m.lockClient.Acquire(ctx, instanceID, workerID)
				if err != nil {
					aborting = true
					log.Error(ctx, "worker failed to acquire lock - aborting test ...", err, logData)
					return
				}

				// Log time it took to acquire (refreshing global min and max), and sleep
				acquireDelay := time.Since(t0)
				t1 := time.Now()

				// Check if we need to abort test (due to some other go-routine having failed)
				if aborting {
					// Unlock
					m.lockClient.Unlock(ctx, lockID)
					log.Info(ctx, "exiting go-routine because the test is being aborted ...", logData)
					return
				}

				// Sleep - represents some  work being done by the caller
				time.Sleep(cfg.SleepTime)

				// Unlock
				t := time.Now()
				m.lockClient.Unlock(ctx, lockID)
				fmt.Printf("\nTime to unlock: %v\n", time.Since(t))

				// calculate time that the lock has been owned
				owningLock := time.Since(t1)

				SetMinMaxTime(acquireDelay)
				log.Info(ctx, "lock has been acquired and released", log.Data{
					"worker_id":                  workerID,
					"time_to_acquire":            acquireDelay.Milliseconds(),
					"time_owning_lock":           owningLock.Milliseconds(),
					"global_max_time_to_acquire": globalMaxAcquireTime.Milliseconds(),
					"global_min_time_to_acquire": globalMinAcquireTime.Milliseconds(),
				})

				if !aborting {
					log.Info(ctx, "worker has finished its work", logData)
				}
				// Sleep corresponds to other actions performed by this caller
				time.Sleep(cfg.SleepTimeBetweenIterations)

				// Success - All the work has been done
			}(fmt.Sprintf("%d", i))
		}
		wg.Wait() // wait for all callers to finish the current iteration
		if aborting {
			return
		}
	}
}

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Connection   *dpMongoDriver.MongoConnection
	URI          string
	healthClient *dpMongoHealth.CheckMongoClient
	lockClient   *dplock.Lock
	IsSSL        bool
	Username     string
	Password     string
	Collection   string
	Database     string
}

// Init creates a new mgo.Session with a strong consistency and a write mode of "majority".
func (m *Mongo) getConnectionConfig(shouldEnableReadConcern, shouldEnableWriteConcern bool) *dpMongoDriver.MongoConnectionConfig {
	return &dpMongoDriver.MongoConnectionConfig{
		IsSSL:                   m.IsSSL,
		ConnectTimeoutInSeconds: connectTimeoutInSeconds,
		QueryTimeoutInSeconds:   queryTimeoutInSeconds,

		Username:                      m.Username,
		Password:                      m.Password,
		ClusterEndpoint:               m.URI,
		Database:                      m.Database,
		Collection:                    m.Collection,
		IsWriteConcernMajorityEnabled: shouldEnableWriteConcern,
		IsStrongReadConcernEnabled:    shouldEnableReadConcern,
	}
}

// Init creates a new mgo.Session with a strong consistency and a write mode of "majortiy"; and initialises the mongo health client.
func (m *Mongo) Init(ctx context.Context, shouldEnableReadConcern, shouldEnableWriteConcern bool, cfg *dplock.ConfigOverride) (err error) {
	if m.Connection != nil {
		return errors.New("Datastor Connection already exists")
	}
	mongoConnection, err := dpMongoDriver.Open(m.getConnectionConfig(shouldEnableReadConcern, shouldEnableWriteConcern))
	if err != nil {
		return err
	}
	m.Connection = mongoConnection

	databaseCollectionBuilder := make(map[dpMongoHealth.Database][]dpMongoHealth.Collection)
	databaseCollectionBuilder[(dpMongoHealth.Database)(Database)] = []dpMongoHealth.Collection{(dpMongoHealth.Collection)(datasetsCollection), (dpMongoHealth.Collection)(editionsCollection), (dpMongoHealth.Collection)(instanceCollection), (dpMongoHealth.Collection)(instanceLockCollection), (dpMongoHealth.Collection)(dimensionOptionsCollection)}

	// Create client and healthclient from session
	client := dpMongoHealth.NewClientWithCollections(m.Connection, databaseCollectionBuilder)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *client,
		Healthcheck: client.Healthcheck,
	}

	// Create MongoDB lock client with the provided config override
	m.lockClient, err = dplock.New(ctx, m.Connection, "instances", cfg)
	return err
}
