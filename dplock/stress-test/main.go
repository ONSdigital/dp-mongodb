package main

import (
	"context"
	"errors"
	"fmt"
	"math"
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
func getLockConfig(oldBehavior bool) *dplock.ConfigOverride {
	if oldBehavior {
		var (
			maxCount               uint          = math.MaxUint32          // this will prevent the 'sleep after maxCount successful acquires' to not be triggered
			acquireMinPeriodMillis uint          = 250                     // old AcquirePeriod = 250 * time.Millisecond
			acquireMaxPeriodMillis uint          = 251                     // old AcquirePeriod = 250 * time.Millisecond
			acquireRetryTimeout    time.Duration = 2500 * time.Millisecond // old: 10 retryes * 250 ms between retries (effectively 2.5s)
			unlockMinPeriodMillis  uint          = 5                       // old var UnlockPeriod = 5 * time.Millisecond
			unlockMaxPeriodMillis  uint          = 6                       // old var UnlockPeriod = 5 * time.Millisecond
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

	// New behavior with default values
	return &dplock.ConfigOverride{}
}

func getMongoDB(ctx context.Context) (*Mongo, error) {
	mongodb := &Mongo{URI: MongoURI}
	if err := mongodb.Init(ctx, getLockConfig(true)); err != nil {
		return nil, err
	}
	log.Info(ctx, "listening to mongo db session", log.Data{"URI": mongodb.URI})
	return mongodb, nil
}

func main() {
	log.Namespace = "dp-mongodb-lock-stress-test"
	ctx := context.Background()

	// Create an array of connections to MongoDB (of size maxInstances)
	m := make([]*Mongo, maxInstances)
	var err error
	for i := 0; i < maxInstances; i++ {
		m[i], err = getMongoDB(ctx)
		if err != nil {
			log.Error(ctx, "failed to initialise dplock", err)
			os.Exit(1)
		}
	}

	// Purge any existing lock before starting the test
	m[0].lockClient.Purger.Purge()

	// default testCfg to be used as a base config for tests
	testCfg := &TestConfig{
		NumCallers:                 2,
		WorkPerCaller:              10,
		SleepTime:                  20 * time.Millisecond,
		SleepTimeBetweenIterations: 250 * time.Millisecond,
	}

	// 2 callers per instance, 1 instances
	testCfg.NumCallers = 2
	log.Info(ctx, "+++ 1. New Test starting +++ 2 callers per instance / 1 instance", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
	t0 := time.Now()
	runTestInstance(ctx, m[0], testCfg, "0")
	t1 := time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 1. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 1. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 10 callers per instance, 1 instances
	testCfg.NumCallers = 10
	log.Info(ctx, "+++ 2. New Test starting +++ 10 callers per instance / 1 instance", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
	t0 = time.Now()
	runTestInstance(ctx, m[1], testCfg, "0")
	t1 = time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 2. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 2. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 2 callers per instance, 2 instances
	testCfg.NumCallers = 2
	log.Info(ctx, "+++ 3. New Test starting +++ 2 callers per instance / 2 instances", log.Data{"test_config": testCfg})
	t0 = time.Now()
	runTestInstances(ctx, []*Mongo{m[2], m[3]}, testCfg)
	t1 = time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 3. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 3. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 10 callers per instance, 2 instances
	testCfg.NumCallers = 10
	log.Info(ctx, "+++ 4. New Test starting +++ 10 callers per instance / 2 instances", log.Data{"test_config": testCfg})
	t0 = time.Now()
	runTestInstances(ctx, []*Mongo{m[4], m[5]}, testCfg)
	t1 = time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 4. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 4. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 2 callers per instance, 6 instances
	testCfg.NumCallers = 2
	log.Info(ctx, "+++ 5. New Test starting +++ 2 callers per instance / 6 instances", log.Data{"test_config": testCfg})
	t0 = time.Now()
	runTestInstances(ctx, m, testCfg)
	t1 = time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 5. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 5. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
	fmt.Print("\n\n\n")

	// 10 callers per instance, 6 instances
	testCfg.NumCallers = 10
	log.Info(ctx, "+++ 6. New Test starting +++ 10 callers per instance / 6 instances", log.Data{"test_config": testCfg})
	t0 = time.Now()
	runTestInstances(ctx, m, testCfg)
	t1 = time.Since(t0)
	m[0].lockClient.Purger.Purge()
	if aborting {
		log.Info(ctx, "=== test 6. [FAILED] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap})
		os.Exit(2)
	}
	log.Info(ctx, "=== test 6. [OK] ===", log.Data{"test_config": testCfg, "usages": m[0].lockClient.Usages.UsagesMap, "test_time": t1.Milliseconds()})
}

// runTestInstances runs multiple instances in parallel, each one running a test with multiple callers
func runTestInstances(ctx context.Context, mongos []*Mongo, cfg *TestConfig) {
	wg := &sync.WaitGroup{}
	for i, mongo := range mongos {
		wg.Add(1)
		go func(serviceID string, m *Mongo) {
			defer wg.Done()
			runTestInstance(ctx, m, cfg, serviceID)
		}(fmt.Sprintf("%d", i), mongo)
	}
	wg.Wait()
}

// runTestInstance runs multiple callers in parallel using the provied Mongo struct
func runTestInstance(ctx context.Context, m *Mongo, cfg *TestConfig, serviceID string) {
	wg := &sync.WaitGroup{}
	instanceID := "testInstance"

	for i := 0; i < cfg.NumCallers; i++ {
		wg.Add(1)
		go func(workerID string) {
			defer wg.Done()
			logData := log.Data{
				"service_id": serviceID,
				"worker_id":  workerID,
			}
			var (
				workDone            int           = 0 // iteration count
				totalTimeAcquire    time.Duration = 0 // accumulation of time delays from lock requesting an acquiring
				totalTimeOwningLock time.Duration = 0 // accumulation of time that a lock has been owned
			)
			for {
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

				// Check if we need to abort test (due to some other go-routine having failed)
				if aborting {
					log.Info(ctx, "exiting go-routine because the test is being aborted ...", logData)
					return
				}

				t1 := time.Now()

				// Log time it took to acquire (refreshing global min and max), and sleep
				acquireDelay := time.Since(t0)
				totalTimeAcquire += acquireDelay
				SetMinMaxTime(acquireDelay)
				log.Info(ctx, "lock has been acquired", log.Data{
					"service_id":                 serviceID,
					"worker_id":                  workerID,
					"time_to_acquire":            acquireDelay.Milliseconds(),
					"global_max_time_to_acquire": globalMaxAcquireTime.Milliseconds(),
					"global_min_time_to_acquire": globalMinAcquireTime.Milliseconds(),
				})
				time.Sleep(cfg.SleepTime)

				// Unlock
				m.lockClient.Unlock(lockID)

				// calculate time that the lock has been owned
				owningLock := time.Since(t1)
				totalTimeOwningLock += owningLock

				// log with total times
				log.Info(ctx, "lock has been released", log.Data{
					"service_id":                    serviceID,
					"worker_id":                     workerID,
					"time_owning_lock":              owningLock.Milliseconds(),
					"total_time_waiting_to_acquire": totalTimeAcquire.Milliseconds(),
					"total_time_owning_a_lock":      totalTimeOwningLock.Milliseconds(),
				})

				workDone++
				if workDone == cfg.WorkPerCaller {
					if !aborting {
						log.Info(ctx, "worker has finished its work", logData)
					}
					return // Success - All the work has been done
				}

				// Check if we need to abort test (due to some other go-routine having failed)
				if aborting {
					log.Info(ctx, "exiting go-routine because the test is being aborted ...", logData)
					return
				}

				// Sleep before next iteration
				time.Sleep(cfg.SleepTimeBetweenIterations)
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
