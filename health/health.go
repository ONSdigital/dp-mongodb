package health

import (
	"context"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
)

var (
	statusDescription = map[string]string{
		healthcheck.StatusOK:       "mongodb is OK",
		healthcheck.StatusCritical: "connection to mongodb failed",
	}

	unixTime = time.Unix(0, 0)
)

// CheckMongoClient is an implementation of the mongo client with a healthcheck
type CheckMongoClient struct {
	client      Client
	healthcheck func(context.Context) (string, error)
}

// Checker calls an api health endpoint and returns a check object to the caller
func (c *CheckMongoClient) Checker(ctx context.Context) (*healthcheck.Check, error) {
	state := healthcheck.StatusOK
	_, err := c.healthcheck(ctx)
	if err != nil {
		state = healthcheck.StatusCritical
	}

	check := getCheck(ctx, c.client.serviceName, state)

	return check, err
}

func getCheck(ctx context.Context, name, state string) (check *healthcheck.Check) {

	currentTime := time.Now().UTC()

	check = &healthcheck.Check{
		Name:        name,
		LastChecked: currentTime,
		LastSuccess: unixTime,
		LastFailure: unixTime,
		Status:      state,
		Message:     statusDescription[state],
	}

	if state == healthcheck.StatusOK {
		check.LastSuccess = currentTime
	} else {
		check.LastFailure = currentTime
	}

	return
}
