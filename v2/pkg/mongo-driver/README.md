# Mongo Driver

This intended to be an abstraction that can encapsulate the mongo db and the document db operations.
All the functionalities are supposed to be done via a common interface.  
This interface is responsible for connection handling as well as the querying.

## Running tests

###  Env Setup

`test` user is expected to be configured with access to `testDb` containing `testCollection`. 

### Against MongoDB
Bring up the mongodb test instance for testing the mongo db suite.

### Against DocumentDB
Forward the documentDb cluster to local and then run the test suites.
This does not work against the document db instance being forwarded.

```
dp ssh develop publishing 4 -- -L 27017:<document-db-cluster-url>:27017
``` 

