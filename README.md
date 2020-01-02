dp-mongodb
================

A mongodb library for DP.

### Getting started

#### Setting up mongo
Using brew, type brew install mongo or the latest binaries can be downloaded [here](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-os-x/#install-mongodb-community-edition-with-homebrew)

#### Running mongo

Follow instructions from mongo db [manual](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-os-x/#run-mongodb)

### health package

Using mongo checker function currently pings a mongo client, further work to check mongo queries based on an applications requirements (level of access and to which databases and collections).

Read the [Health Check Specification](https://github.com/ONSdigital/dp/blob/master/standards/HEALTH_CHECK_SPECIFICATION.md) for details.

Instantiate a mongo health checker
```
import mongoHealth "github.com/ONSdigital/dp-mongo/health"

...

    mongoClient := mongoHealth.NewClient(<mgo session>)

    mongoHealth := mongoHealth.CheckClient{
        client: mongoClient,
    }
...
```

Call mongo health checker with `mongoHealth.Checker(context.Background())` and this will return a check object like so:

```json
{
    "name": "string",
    "status": "string",
    "message": "string",
    "last_checked": "ISO8601 - UTC date time",
    "last_success": "ISO8601 - UTC date time",
    "last_failure": "ISO8601 - UTC date time"
}
```

### Configuration

Configuration of the health check takes place via arguments passed to the `.Create()` function

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2020, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
