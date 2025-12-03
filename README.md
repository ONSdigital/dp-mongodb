# dp-mongodb

A mongodb library for DP.

## Getting started

### Setting up mongo

Using brew, type brew install mongo or the latest binaries can be downloaded [here](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-os-x/#install-mongodb-community-edition-with-homebrew)

### Running mongo

Follow instructions from mongo db [manual](https://docs.mongodb.com/manual/tutorial/install-mongodb-on-os-x/#run-mongodb)

## health package

The mongo checker function currently pings the mongo client, and checks that all collections given when the checker was created, actually exist.

Read the [Health Check Specification](https://github.com/ONSdigital/dp/blob/master/standards/HEALTH_CHECK_SPECIFICATION.md) for details.

Instantiate a mongo health checker

```go
import mongoHealth "github.com/ONSdigital/dp-mongo/health"
import mongoDriver "github.com/ONSdigital/dp-mongo/mongodb"

...

    healthClient := mongoHealth.NewClientWithCollections(<mongoDriver.MongoConnection>, <map[mongoHealth.Database][]mongoHealth.Collection>)

...
```

Calling mongo health checker: `healthClient.Checker(context.Context, *healthcheck.CheckState)` will fill out the check object like so:

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

## Configuration

Configuration of the health check takes place via arguments passed to the `NewClient() or NewClientWithCollections()` functions

## Tools

To run some of our tests you will need additional tooling:

### Audit

We use `dis-vulncheck` to do auditing, which you will [need to install](https://github.com/ONSdigital/dis-vulncheck).

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

## License

Copyright © 2024, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.
