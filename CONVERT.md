# Converting from MongoDB v1 to v3
To convert existing v1 code to v3 complete the following steps

> Note you will need to pass the context (ctx) into any existing functions

## 1. Configuration for Connecting to MongoDB / DocumentDB

```go
type MongoDriverConfig struct {
	Username        string                          // The remote database username
	Password        string                          // The remote database password
	ClusterEndpoint string                          // The endpoint
	Database        string                          // The hosted database 
	Collections                   map[string]string // TODO
	ReplicaSet                    string            // TODO  
	IsStrongReadConcernEnabled    bool              // TODO
	IsWriteConcernMajorityEnabled bool              // TODO
	ConnectTimeout time.Duration                    // TODO
	QueryTimeout   time.Duration                    // TODO

	TLSConnectionConfig
}

type TLSConnectionConfig struct {
	IsSSL              bool                         // Set to true
	VerifyCert         bool                         // Set to true
	CACertChain        string                       // TODO
	RealHostnameForSSH string                       // TODO
}
```

## 2. Connection to MongoDB / DocumentDB

### V1 code
```go
func (m *Mongo) Init() (session *mgo.Session, err error) {
	if session != nil {
		return nil, errors.New("session already exists")
	}

	if session, err = mgo.Dial(m.URI); err != nil {
		return nil, err
	}

	session.EnsureSafe(&mgo.Safe{WMode: "majority"})
	session.SetMode(mgo.Strong, true)
	return session, nil
}
```
### V3 code
```go
func Init(cfg mongodriver.MongoDriverConfig) (m *Mongo, err error) {

	m, err = mongodriver.Open(&cfg)
	if err != nil {
		return nil, err
	}

    return m
}
```

## 3. Convert Get Offset of Documents

### V1 code
```go
func (m *Mongo) GetItems(ctx context.Context, offset int, limit int) (*results, error) {
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

	var items []*models.Items
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

	return &items
}
```
### V3 code
> Note you must pass a sort element as DocumentDB does not support the [default ordering of results](https://docs.aws.amazon.com/documentdb/latest/developerguide/functional-differences.html#functional-differences.result-ordering)
```go
func (m *Mongo) GetWithOffset(ctx context.Context, offset int, limit int) (*results, error) {
	var items []*models.Items
	totalCount, err := m.connection.Collection(CollectionName).Find(ctx, bson.D{}, &items, mongodriver.Sort(bson.M{"_id": 1}), mongodriver.Offset(offset), mongodriver.Limit(limit))
	if err != nil {
		log.Error(ctx, "error finding items", err)
		return nil, err
	}

	return items
}
```

## 4. Convert Get Item
### V1 code
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
### V3 code
```go
func (m *Mongo) GetItem(ctx context.Context, id string) (*models.Item, error) {
	var item models.Item
	err := m.connection.Collection(CollectionName).FindOne(ctx, bson.M{"_id": id}, &item)
	if err != nil {
		if errors.Is(err, mongodriver.ErrNoDocumentFound) {
			return nil, errs.ErrItemNotFound
		}
		return nil, err
	}

	return &item, nil
}
```
## 5. Convert Update / Insert Item
### V1 code
```go
func (m *Mongo) AddIem(item *models.Item) error {
	s := m.Session.Copy()
	defer s.Close()
	_, err := s.DB(m.Database).C(m.Collection).UpsertId(item.ID, item)
	return err
}
```
### V3 code
```go
func (m *Mongo) AddItem(ctx context.Context, item models.Item) error {
	_, err := m.connection.Collection(CollectionName).UpsertById(ctx, item.ID, bson.M{"$set": item})

	return err
}
```