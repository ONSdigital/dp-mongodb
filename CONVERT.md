## Converting from MongoDB v1 to v3
In moving from v1 to v3 certain functionality is explicitly addressed
### 1. Changing the driver
   1. This involved moving from the globalsign mgo driver: 'github.com/globalsign/mgo', to the official golang driver: 'go.mongodb.org/mongo-driver'. For the most part this change is hidden in the library, but where the driver is imported by the application (often to access the 'bson' package), the new driver package will have to be imported. This is usually a straight substitution in the code, for example from 'github.com/globalsign/mgo/bson' to 'go.mongodb.org/mongo-driver/bson'. Please check your go.mod/go.sum files to ensure all references to the original globalsign mgo driver have been removed.
   2. As part of the upgrade to the new driver, the library's functions and methods were upgraded to take a context.Context. Where necessary, a context will have to be passed along the function chain which ends in the call to library.
### 2. Changing the config and means of initialisation.
   The library now defines an explicit configuration that must be passed by an application. This is:
```go
type MongoDriverConfig struct {
   Username                      string            // The remote database username
   Password                      string            // The remote database password
   ClusterEndpoint               string            // The endpoint
   Database                      string            // The hosted database
   Collections                   map[string]string // A mapping from a collection's 'Well Known Name' to 'Actual Name'
   ReplicaSet                    string            // A name for the DocumentDB replica set, the empty string for non DocumentDB clusters
   IsStrongReadConcernEnabled    bool              // Whether a read value must be acknowledged by a majority of servers in the cluster to be returned
   IsWriteConcernMajorityEnabled bool              // Whether a value to be written must be acknowledged by a majority of servers in the cluster before returning
   ConnectTimeout                time.Duration     // Default timeout value to connect to a server
   QueryTimeout                  time.Duration     // Default timeout value for a query

	TLSConnectionConfig
}

type TLSConnectionConfig struct {
   IsSSL              bool                         // Whether to use TLS in connections to the server (required to be true for the production environment)
   VerifyCert         bool                         // When IsSSL is true, whether to validate the server's TLS certificate (required to be true for the production environment)
   CACertChain        string                       // The (chain of) Certificate Authority certificate(s) to be used to validate the server's certificate (must be supplied when VerifyCert is true)
   RealHostnameForSSH string                       // When using ssh to proxy to a server, as is often the case when testing locally, this can be set to the DNS name of
                                                   // the actual server, since with ssh the server name will appear to be 'localhost'
}
```
As a convenience to the applications the MongoDriverConfig struct has envconfig annotations that allow the application to use the MongoDriverConfig directly. An application is, of course, free to define its config in any way it chooses, provided it passes an instance of the above struct to the initialisation
```go
type MongoDriverConfig struct {
	Username                      string            `envconfig:"MONGODB_USERNAME"    json:"-"`
	Password                      string            `envconfig:"MONGODB_PASSWORD"    json:"-"`
	ClusterEndpoint               string            `envconfig:"MONGODB_BIND_ADDR"   json:"-"`
	Database                      string            `envconfig:"MONGODB_DATABASE"`
	Collections                   map[string]string `envconfig:"MONGODB_COLLECTIONS"`
	ReplicaSet                    string            `envconfig:"MONGODB_REPLICA_SET"`           // The standard default value for our production environment is 'rs0'
	IsStrongReadConcernEnabled    bool              `envconfig:"MONGODB_ENABLE_READ_CONCERN"`   // The standard default value for our production environment is false
	IsWriteConcernMajorityEnabled bool              `envconfig:"MONGODB_ENABLE_WRITE_CONCERN"`  // The standard default value for our production environment is true
	ConnectTimeout                time.Duration     `envconfig:"MONGODB_CONNECT_TIMEOUT"`       // The standard default value for our production environment is 5*time.Second - this is expressed as '5s' in the configuration file
	QueryTimeout                  time.Duration     `envconfig:"MONGODB_QUERY_TIMEOUT"`         // The standard default value for our production environment is 15*time.Second - this is expressed as '15s' in the configuration file

	TLSConnectionConfig
}

type TLSConnectionConfig struct {
   IsSSL              bool   `envconfig:"MONGODB_IS_SSL"`
   VerifyCert         bool   `envconfig:"MONGODB_VERIFY_CERT"`
   CACertChain        string `envconfig:"MONGODB_CERT_CHAIN"`
   RealHostnameForSSH string `envconfig:"MONGODB_REAL_HOSTNAME"`
}
```
The initialisation code has changed as follows (taken from dp-dataset-api):
#### V1 code
```go
type Mongo struct {
   CodeListURL    string
   Collection     string
   Database       string
   DatasetURL     string
   Session        *mgo.Session
   URI            string
   lastPingTime   time.Time
   lastPingResult error
   healthClient   *dpMongoHealth.CheckMongoClient
   lockClient     *dpMongoLock.Lock
}

const (
   editionsCollection     = "editions"
   instanceCollection     = "instances"
   instanceLockCollection = "instances_locks"
   dimensionOptions       = "dimension.options"
)

// Init creates a new mgo.Session with a strong consistency and a write mode of "majortiy"; and initialises the mongo health client.
func (m *Mongo) Init(ctx context.Context) (err error) {
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
	databaseCollectionBuilder[(dpMongoHealth.Database)(m.Database)] = []dpMongoHealth.Collection{(dpMongoHealth.Collection)(m.Collection), (dpMongoHealth.Collection)(editionsCollection), (dpMongoHealth.Collection)(instanceCollection), (dpMongoHealth.Collection)(instanceLockCollection), (dpMongoHealth.Collection)(dimensionOptions)}

	// Create client and healthclient from session
	client := dpMongoHealth.NewClientWithCollections(m.Session, databaseCollectionBuilder)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *client,
		Healthcheck: client.Healthcheck,
	}

	// Create MongoDB lock client, which also starts the purger loop
	m.lockClient, err = dpMongoLock.New(ctx, m.Session, m.Database, instanceCollection, nil)
	if err != nil {
		return err
	}
	return nil
}
```
#### V3 code
```go
type Mongo struct {
   config.MongoConfig
   
   Connection   *mongodriver.MongoConnection
   healthClient *mongohealth.CheckMongoClient
   lockClient   *mongolock.Lock
}

// Init returns an initialised Mongo object encapsulating a connection to the mongo server/cluster with the given configuration,
// a health client to check the health of the mongo server/cluster, and a lock client
func (m *Mongo) Init(ctx context.Context) (err error) {
	m.Connection, err = mongodriver.Open(&m.MongoDriverConfig)
	if err != nil {
		return err
	}

	databaseCollectionBuilder := map[mongohealth.Database][]mongohealth.Collection{
		(mongohealth.Database)(m.Database): {
			mongohealth.Collection(m.ActualCollectionName(config.DatasetsCollection)),
			mongohealth.Collection(m.ActualCollectionName(config.EditionsCollection)),
			mongohealth.Collection(m.ActualCollectionName(config.InstanceCollection)),
			mongohealth.Collection(m.ActualCollectionName(config.DimensionOptionsCollection)),
			mongohealth.Collection(m.ActualCollectionName(config.InstanceLockCollection)),
		},
	}
	m.healthClient = mongohealth.NewClientWithCollections(m.Connection, databaseCollectionBuilder)
	m.lockClient = mongolock.New(ctx, m.Connection, m.ActualCollectionName(config.InstanceCollection))

	return nil
}
```
### 3. Unify on one mechanism to access a collection
Remove the use of Connection.GetConfiguredCollection(), and rename Connection.C() to Connection.Collection()
#### V1 code
```go
func (m *Mongo) GetDataset(id string) (*models.DatasetUpdate, error) {
	s := m.Session.Copy()
	defer s.Close()
	var dataset models.DatasetUpdate
	err := s.DB(m.Database).C("datasets").Find(bson.M{"_id": id}).One(&dataset)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, errs.ErrDatasetNotFound
		}
		return nil, err
	}

	return &dataset, nil
}
```
#### V3 code
```go
func (m *Mongo) GetDataset(ctx context.Context, id string) (*models.DatasetUpdate, error) {
	var dataset models.DatasetUpdate
	err := m.Connection.Collection(m.ActualCollectionName(config.DatasetsCollection)).FindOne(ctx, bson.M{"_id": id}, &dataset)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocumentFound) {
			return nil, errs.ErrDatasetNotFound
		}
		return nil, err
	}

	return &dataset, nil
}

```
Note the use of the standard errors.Is() mechanism, part of the error handling simplification in V3
### 4. Making search/find easier
This was achieved with a number of changes:
   1. Removing the use of an explicit Find object in the library, whose primary purpose was to enable the use of a builder pattern to define the parameters of the search
   2. Introduce a simple and standard options handling, whereby a find operation on a collection would take one or more find options: Sort, Offset, Limit, Project (see options.go in the library's mongodb package)
   3. Remove the Iterator object from the library, and provide a simple dp-mongodblib.Collection.FindOne() method and a dp-mongodblib.Collection.Find() method. The changes between V1 and V3 are:
#### V1 code to find all documents
```go
func (m *Mongo) GetItems(ctx context.Context, offset int, limit int) ([]models.Items, error) {
	s := m.Session.Copy()
	defer s.Close()

	query := s.DB(m.Database).C(m.Collection).Find(nil)
	totalCount, err := query.Count()
	if err != nil {
		if err == mgo.ErrNotFound {
			return emptyItemsResults(offset, limit), nil
		}
		log.Event(ctx, "error counting items", log.ERROR, log.Error(err))
		return nil, err
	}

	var items []models.Items
	if limit > 0 {
		iter := query.Sort().Skip(offset).Limit(limit).Iter()
		defer func() {
			err := iter.Close()
			if err != nil {
				log.Event(ctx, "error closing job iterator", log.ERROR, log.Error(err), log.Data{})
			}
		}()

		if err := iter.All(&items); err != nil {
			return nil, err
		}
	}

	return items
}
```
#### V3 code to find all documents
> Note you must pass a sort option as DocumentDB does not support the [default ordering of results](https://docs.aws.amazon.com/documentdb/latest/developerguide/functional-differences.html#functional-differences.result-ordering)
```go
func (m *Mongo) GetItems(ctx context.Context, offset, limit int) ([]models.Items, totalCount int, err error) {
    var items []models.Items
	totalCount, err = m.Connection.Collection(m.ActualCollectionName(config.ItemsCollection)).
		Find(ctx, bson.M{"current": bson.M{"$exists": true}}, &items,
		options.Sort(bson.M{"_id": -1}), options.Offset(offset), options.Limit(limit))
	if err != nil {
		return nil, 0, err
	}

	return items, totalCount, nil
}
```
#### V1 code to find one document
```go
func (m *Mongo) GetItem(id string) (*models.Item, error) {
	s := m.Session.Copy()
	defer s.Close()
	var item models.Item
	err := s.DB(m.Database).C(m.Collection).Find(bson.M{"_id": id}).One(&item)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, errs.ErrItemNotFound
		}
		return nil, err
	}

	return &item, nil
}
```
#### V3 code to find one document
```go
func (m *Mongo) GetItem(ctx context.Context, id string) (*models.Item, error) {
	var item models.Item
	err := m.connection.Collection(m.ActualCollectionName(config.ItemsCollection)).FindOne(ctx, bson.M{"_id": id}, &item)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocumentFound) {
			return nil, errs.ErrItemNotFound
		}
		return nil, err
	}

	return &item, nil
}
```

### 5. Converting Update / Insert Item
#### V1 code
```go
func (m *Mongo) AddIem(item *models.Item) error {
	s := m.Session.Copy()
	defer s.Close()
	_, err := s.DB(m.Database).C(m.Collection).UpsertId(item.ID, item)
	return err
}
```
#### V3 code
```go
func (m *Mongo) AddItem(ctx context.Context, item *models.Item) error {
	_, err := m.connection.Collection(m.ActualCollectionName(config.ItemsCollection)).UpsertById(ctx, item.ID, bson.M{"$set": item})

	return err
}
```

### 6. Transactions
A small but powerful transaction api has been added to mongo v3. It is well documented in the Example and Tests in the transaction_test.go file, but a summary is as follows:<br>
The existing MongoConnection object now has a new RunTransaction method:
```go
    type TransactionFunc func(transactionCtx context.Context) (interface{}, error)

    func (ms *MongoConnection) RunTransaction(ctx context.Context, withRetries bool, fn TransactionFunc) (interface{}, error) {....}
```
The RunTransaction method starts the transaction and calls the provided TransactionFunc fn<br>
The transactionCtx is the context in which the transaction is to be executed. Any mongo operation executed with that context (by passing the transactionCtx to
the given mongo operation) will occur within the transaction in an acid fashion). If the TransactionFunc returns an error the transaction is rolled back;
if not, the transaction is committed. In either case the original return value and error (from TransactionFunc), are returned to the caller of the RunTransaction method.
If an internal error occurs when committing/rolling back the transaction, this error is returned by RunTransaction, and the returned value should not be trusted. <br>
There is one exception to this: when the value of withRetries is true, and a ‘transient transaction error’ occurs on committal, the transaction will be re-tried, i.e. 
the TransactionFunc will be re-run.
```go
func example() {
        .
        .
			
        // conn is a MongoConnection to a replica set cluster
		
        r, e := conn.RunTransaction(ctx, true, func(transactionCtx context.Context) (interface{}, error) {
                    var obj AnObjectType
                    err := conn.Collection(collection-name).FindOne(transactionCtx, bson.M{"_id": AnIdentifier}, &obj)
                    if err != nil {
                        return nil, fmt.Errorf("could not find object in collection (%s): %w", collection-name, err)
                    }
                    
                    if obj.SomeStateVariable != "What I Expect" {
                        return nil, badObjectState
                    }
    
                    obj.SomeStateVariable = "Updated Value"
                    _, err = conn.Collection(collection-name).Update(transactionCtx, bson.M{"_id": 1}, bson.M{"$set": obj})
                    if err != nil {
                        return nil, fmt.Errorf("could not write object in collection (%s): %w", collection-name, err)
                    }
                    
                    return obj, nil
                })
    
        switch {
        // handle this special case, where we have aborted the transaction because the object was not in a valid state
        case errors.Is(e, badObjectState):
    
        // otherwise, a runtime error, i.e. couldn't complete the transaction for some other reason (even with retries)
        case !errors.Is(e, nil):
    
        // transaction completed successfully, and r contains a valid object
        default:
            if _, ok := r.(AnObjectType); !ok {
                // This should not be possible :-)
            }
        }
}
```
##### When is a transaction needed
Generally when multiple writes are needed atomically across multiple documents in the same or different collections.
##### When is a transaction NOT needed (the dplock package)
There has been extensive use of the dp-mongodb/dplock package to lock an object for ’read for write’ operations, i.e. we lock an object, read its value, check it’s state, 
and in certain cases perform an update write (as in the example above). This package was most likely developed before mongo had implemented transactions, and its use is 
computationally expensive, so using the new transaction mechanism is preferred over the use of the dplock package. <br>
Having said that, the 'read for write' pattern is a very standard pattern and should not need the use of locking or transactions. This pattern is exactly what ETags 
were developed for. So instead of the above, something like the following can be used:
```go
func exampleUpdate(ctx context.Context, obj AnObjectType) error {
        .
        .
			
        // conn is a MongoConnection to a replica set cluster
		
        var existing AnObjectType
        err := conn.Collection(collection-name).FindOne(transactionCtx, bson.M{"_id": obj.ID}, &existing)
        if err != nil {
            return fmt.Errorf("could not find object in collection (%s): %w", collection-name, err)
        }
        
        if obj.ETag != existing.ETag {
            return badObjectVersion
        }

        // Check state changes are okay

        obj.Etag = calculateNewEtag(obj, existing)
        // Use the existing ETag value to ensure we only update if the object has not changed state since we read it above
        _, err = conn.Collection(collection-name).Update(transactionCtx, bson.M{"_id": obj.ID, "e_tag": existing.ETag}, bson.M{"$set": obj})
        if err != nil {
            return fmt.Errorf("could not write object in collection (%s): %w", collection-name, err)
        }
        
        return nil
}
```
Etags should be used extensively. If for some reason Etags are not implemented for an object type, there is no alternative other than to use a transaction.
#### Testing Transactions
Transactions are only available with mongo clusters in a replica set. All ONS production systems of MongoDB/DocumentDB run as replica sets.<br>
To allow for testing transactions with a replica set, versions of dp-mongodb-in-memory >= Release 1.5.0 have the ability to start a mongo server as a replica set - see [StartWithReplicaSet()](https://github.com/ONSdigital/dp-mongodb-in-memory/blob/8c15e7b214955795920e49dc1db496daf6b8078c/main.go#L46) or [StartWithOption()](https://github.com/ONSdigital/dp-mongodb-in-memory/blob/8c15e7b214955795920e49dc1db496daf6b8078c/main.go#L70) <br>
Versions of dp-component-test >= 0.9.0 use the new version of dp-mongodb-in-memory, and using the standard [NewMongoFeature(mongoOptions MongoOptions)](https://github.com/ONSdigital/dp-component-test/blob/3cdf30c6782e872d45ca55d41a9d9f030209a776/mongo_feature.go#L47) where [MongoOptions.ReplicaSetName](https://github.com/ONSdigital/dp-component-test/blob/3cdf30c6782e872d45ca55d41a9d9f030209a776/mongo_feature.go#L29) has a non-empty set name,
results in the feature using a mongo server set up as a replica set of the given name.
#### Long-running Transactions
The handling of an executing transaction at service shutdown proceeds as one might expect. If a transaction is in progress when the service receives a shutdown signal, 
the graceful shutdown process commences:  
- If the transaction finishes before the graceful shutdown period ends (generally 5 seconds), the transaction ends successfully (either commits or rollsback according to the business logic), the mongo connection is successfully closed, and the shutdown ends gracefully.  
- If the transaction does not finish before the graceful period ends, then service is terminated ‘ungracefully’, i.e. after the graceful shutdown period expires, the
server terminates with an error code, the client will receive an http error, and the mongo server will rollback the transaction.  
### 7. Setting up a DocumentDB database and collections in AWS
Before setting up the required database and collections for v3 (Document DB), in AWS, you need to have ready the following values:
1. Database name
2. Names of any collections that should exist in the database
3. Database username
4. Database password

Once those values are to hand then they can be used in updating the following documents:
1. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/group_vars/docdb
2. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/inventories/sandbox/group_vars/all
3. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/inventories/prod/group_vars/all

Once those documents have been updated, and the changes approved and merged, then ansible should be run. This will apply the relevant changes to AWS.

### 8. Updating the secrets in sandbox and prod
The secrets (for the relevant service) in dp-configs, for sandbox and prod, need to contain all the attributes that are in the MongoDriverConfig struct.
