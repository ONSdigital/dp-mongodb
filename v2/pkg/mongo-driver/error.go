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

type ErrDisconnect struct {
	MongoError
}

type ErrTimeout struct {
	MongoError
}

type ErrCollectionNotFound struct {
	MongoError
}

type ErrServerError struct {
	MongoError
}

func wrapMongoError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, mongo.ErrClientDisconnected) {
		return &ErrDisconnect{MongoError{"Client Disconnect", err}}
	}

	if mongo.IsTimeout(err) {
		return &ErrTimeout{MongoError{"Timeout", err}}
	}

	var commandError *mongo.CommandError

	// should not fail, but return generic command error it does
	if errors.As(err, &commandError) {
		if commandError.Code == 26 || commandError.Message == "ns not found" {
			return &ErrCollectionNotFound{MongoError{"Collection not found", err}}
		}
	}

	return &ErrServerError{MongoError{"Server Error", err}}
}

func IsErrCollectionNotFound(err error) bool {
	var collectionNotFound *ErrCollectionNotFound

	return errors.As(err, &collectionNotFound)

}
