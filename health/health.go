package health

import (
	"context"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
)

var (
	healthyMessage = "mongodb is OK"
)

// Healthcheck health check function
type Healthcheck = func(context.Context) (string, error)

// CheckMongoClient is an implementation of the mongo client with a healthcheck
type CheckMongoClient struct {
	client      Client
	healthcheck Healthcheck
}

// Checker calls an api health endpoint and returns a check object to the caller
func (c *CheckMongoClient) Checker(ctx context.Context) (*healthcheck.Check, error) {
	_, err := c.healthcheck(ctx)
	currentTime := time.Now().UTC()
	c.client.Check.LastChecked = &currentTime
	if err != nil {
		c.client.Check.LastFailure = &currentTime
		c.client.Check.Status = healthcheck.StatusCritical
		c.client.Check.Message = err.Error()
		return c.client.Check, err
	}
	c.client.Check.LastSuccess = &currentTime
	c.client.Check.Status = healthcheck.StatusOK
	c.client.Check.Message = healthyMessage
	return c.client.Check, nil
}
