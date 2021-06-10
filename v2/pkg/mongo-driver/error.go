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

type ErrNoDocumentFound struct {
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

	if err == mongo.ErrNoDocuments {
		return NewErrNoDocumentFoundError("No document found", err)
	}

	return &ErrServerError{MongoError{"Server Error", err}}
}

func IsErrNoDocumentFound(err error) bool {
	var noDocumentFound *ErrNoDocumentFound

	return errors.As(err, &noDocumentFound)
}

func IsServerErr(err error) bool {
	var serverError *ErrServerError

	return errors.As(err, &serverError)
}

func NewErrNoDocumentFoundError(msg string, err error) error {
	return &ErrNoDocumentFound{MongoError{msg, err}}
}
