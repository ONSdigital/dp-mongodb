package health

import (
	"context"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/log"
	mgo "github.com/globalsign/mgo"
)

// ServiceName mongodb
const ServiceName = "mongodb"

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

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo       *mgo.Session
	serviceName string
	Check       *health.Check
}

// NewClient returns a new health check client using the given service
func NewClient(db *mgo.Session) *Client {

	// Initial Check struct
	check := &health.Check{Name: ServiceName}

	// Create Client
	return &Client{
		mongo:       db,
		serviceName: ServiceName,
		Check:       check,
	}
}

// Healthcheck calls service to check its health status
func (m *Client) Healthcheck(ctx context.Context) (res string, err error) {
	s := m.mongo.Copy()
	defer s.Close()
	res = m.serviceName
	err = s.Ping()
	if err != nil {
		log.Event(ctx, "Ping mongo", log.Error(err))
	}

	return
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
