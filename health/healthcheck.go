package health

import (
	"context"

	"github.com/ONSdigital/log.go/log"
	mgo "github.com/globalsign/mgo"
	health "github.com/ONSdigital/dp-healthcheck/healthcheck"

)

// ServiceName mongodb
const ServiceName = "mongodb"

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo        *mgo.Session
	serviceName  string
	Check        *health.Check
}

// NewClient returns a new health check client using the given service
func NewClient(db *mgo.Session) *Client {

	// Initial Check struct
	check := &health.Check{Name: ServiceName}

	// Create Client
	return &Client{
		mongo:       db,
		serviceName: ServiceName,
		Check: check,
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
