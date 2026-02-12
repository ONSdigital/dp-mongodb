# Mongo Driver

This library is intended to be an abstraction that can encapsulate the mongo db and the document db operations.
All functionality that accesses the db is intended to be accessed via this library.
The library is responsible for connection handling as well as querying.

## How to Connect to DocumentDB

There are now two different ways to connect to DocumentDB; the legacy way and the EKS way. 

Both use the same username and password values but there are some extra parameters added to the EKS connection string, as follows:

Legacy connection string format:

```go
mongodb://<username>:<password>@<cluster endpoint>/<name of database>
```

EKS connection string format:

```go
mongodb://<username>:<password>@<cluster endpoint>/<name of database>?tls=true&replicaSet=rs0&readpreference=secondaryPreferred
```

However, as well as the extra parameters, the EKS connection makes use of an AWS CA (Certificate Authority) file named "global-bundle.pem".
The CA file gets downloaded on the fly, using wget, from the following link:

"https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"

To specify an EKS connection, set the following environment variable to true:

```shell
export CONNECT_EKS=true
```

By default, CONNECT_EKS is set to false, which means a legacy connection will be used.

To test the connection you need to provide the relevant values for:

- MONGODB_USERNAME
- MONGODB_PASSWORD
- MONGODB_BIND_ADDR (this is the cluster endpoint)
- MONGODB_DATABASE

Then you can connect using the following code or similar:

```go
	connection, err := mongodriver.Open(ctx, &mongoDriverConfig)
	if err != nil {
		return err
	}

	err = connection.Ping(ctx, timeOutInSeconds)
	if err != nil {
		log.Error(ctx, "Ping mongo", err)
		return err
	}
	log.Info(ctx, "Pong from mongo test!")
```

There's an example of a mongo test like this in the [dis-test-eks-app](https://github.com/ONSdigital/dis-test-eks-app)
