package health

import (
	"context"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
)

var (
	healthyMessage = "mongodb is OK"

	unixTime = time.Unix(0, 0)
)

type Healthcheck = func(context.Context) (string, error)

// CheckMongoClient is an implementation of the mongo client with a healthcheck
type CheckMongoClient struct {
	client      Client
	healthcheck Healthcheck
}

// Checker calls an api health endpoint and returns a check object to the caller
func (c *CheckMongoClient) Checker(ctx context.Context) (*healthcheck.Check, error) {
	state := healthcheck.StatusOK
	message := healthyMessage
	_, err := c.healthcheck(ctx)
	if err != nil {
		message = err.Error()
		state = healthcheck.StatusCritical
	}

	check := getCheck(ctx, c.client.serviceName, state, message)

	return check, err
}

func getCheck(ctx context.Context, name, state, message string) (check *healthcheck.Check) {

	currentTime := time.Now().UTC()

	check = &healthcheck.Check{
		Name:        name,
		LastChecked: currentTime,
		LastSuccess: unixTime,
		LastFailure: unixTime,
		Status:      state,
		Message:     message,
	}

	if state == healthcheck.StatusOK {
		check.LastSuccess = currentTime
	} else {
		check.LastFailure = currentTime
	}

	return
}
