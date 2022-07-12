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

### 6. Setting up a DocumentDB database and collections in AWS
Before setting up the required database and collections for v3 (Document DB), in AWS, you need to have ready the following values:
1. Database name
2. Names of any collections that should exist in the database
3. Database username
4. Database password

Once those values are to hand then they can be used in updating the following documents:
1. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/group_vars/docdbhttps://github.com/ONSdigital/dp-setup/blob/awsb/ansible/group_vars/docdb
2. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/inventories/sandbox/group_vars/all
3. https://github.com/ONSdigital/dp-setup/blob/awsb/ansible/inventories/prod/group_vars/all

Once those documents have been updated, and the changes approved and merged, then ansible should be run. This will apply the relevant changes to AWS.

For instructions on how to do the above tasks please see dp-setup.

### 7. Updating the secrets in sandbox and prod
The secrets in dp-configs, for sandbox and prod, need to contain all the attributes that are in the MongoDriverConfig struct.
