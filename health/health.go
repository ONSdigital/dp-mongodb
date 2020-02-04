package health

import (
	"context"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/log"
	mgo "github.com/globalsign/mgo"
)

// ServiceName mongodb
const ServiceName = "mongodb"

var (
	// HealthyMessage is the message that will be used in healthcheck when mongo is Healthy
	HealthyMessage = "mongodb is OK"
)

// Healthcheck health check function
type Healthcheck = func(context.Context) (string, error)

// CheckMongoClient is an implementation of the mongo client with a healthcheck
type CheckMongoClient struct {
	Client      Client
	Healthcheck Healthcheck
}

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo       *mgo.Session
	serviceName string
}

// NewClient returns a new health check client using the given service
func NewClient(db *mgo.Session) *Client {
	return &Client{
		mongo:       db,
		serviceName: ServiceName,
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

// Checker calls an api health endpoint and  updates the provided CheckState accordingly
func (c *CheckMongoClient) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	_, err := c.Healthcheck(ctx)
	if err != nil {
		state.Update(healthcheck.StatusCritical, err.Error(), 0)
		return nil
	}
	state.Update(healthcheck.StatusOK, HealthyMessage, 0)
	return nil
}
