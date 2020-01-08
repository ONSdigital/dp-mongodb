package health

import (
	"context"

	"github.com/ONSdigital/log.go/log"
	mgo "github.com/globalsign/mgo"
)

// Client provides a healthcheck.Client implementation for health checking the service
type Client struct {
	mongo       *mgo.Session
	serviceName string
}

// NewClient returns a new health check client using the given service
func NewClient(db *mgo.Session) *Client {
	return &Client{
		mongo:       db,
		serviceName: "mongodb",
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
