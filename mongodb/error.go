package mongodb

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrDisconnect      = mongo.ErrClientDisconnected
	ErrNoDocumentFound = mongo.ErrNoDocuments
)

type Error struct {
	inner error
}

func (e Error) Error() string {
	return e.inner.Error()
}

func (e Error) Unwrap() error {
	return e.inner
}

func IsTimeout(err error) bool {
	e, ok := err.(Error)
	return ok && mongo.IsTimeout(e.inner)
}

func IsServerErr(err error) bool {
	e, ok := err.(Error)
	return ok && !mongo.IsTimeout(e.inner)
}

func wrapMongoError(err error) error {
	if err == nil {
		return nil

	}

	switch {
	case errors.Is(err, mongo.ErrClientDisconnected):
		return ErrDisconnect
	case errors.Is(err, mongo.ErrNoDocuments):
		return ErrNoDocumentFound
	default:
		return Error{inner: err}
	}

}
