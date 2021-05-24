package mongo_driver

import (
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoError struct {
	Reason string
	Inner  error
}

func (m *MongoError) Error() string {
	if m.Inner == nil {
		return fmt.Sprintf("%s", m.Reason)
	}

	return fmt.Sprintf("%s caused by %v", m.Reason, m.Inner)
}

func (m *MongoError) Unwrap() error {
	return m.Inner
}

type Disconnect struct {
	MongoError
}

type Timeout struct {
	MongoError
}

type CollectionNotFound struct {
	MongoError
}

type ServerError struct {
	MongoError
}

func wrapMongoError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, mongo.ErrClientDisconnected) {
		return &Disconnect{MongoError{"Client Disconnect", err}}
	}

	if mongo.IsTimeout(err) {
		return &Timeout{MongoError{"Timeout", err}}
	}

	var commandError *mongo.CommandError

	// should not fail, but return generic command error it does
	if errors.As(err, &commandError) {
		if commandError.Code == 26 || commandError.Message == "ns not found" {
			return &CollectionNotFound{MongoError{"Collection not found", err}}
		}
	}

	return &ServerError{MongoError{"Server Error", err}}
}
