package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// TransactionFunc is the type signature of a client function that is to be executed within a transaction,
// defined by the transaction context provided as the parameter to the function - transactionCtx
// All calls to the mongodb library to be executed within the transaction must be passed the transactionCtx
// Returning an error from the function indicates the transaction is to be aborted
type TransactionFunc func(transactionCtx context.Context) (interface{}, error)

// RunTransaction will execute the given function - fn - within a transaction, defined by the transaction context - transactionCtx
// If withRetries is true, the transaction will retry on a transient transaction error - this can be due to a network
// error, but may also occur if the state of an object being updated in the transaction has been changed since the transaction
// started. This latter behaviour may or may be suitable depending on the circumstances, and so is optional
// The return values of the function are the return values provided by the TransactionFunc fn, except in the case where
// runtime errors occur outside the TransactionFunc fn, when committing or aborting the transaction
func (ms *MongoConnection) RunTransaction(ctx context.Context, withRetries bool, fn TransactionFunc) (interface{}, error) {
	opts := options.Session().
		SetCausalConsistency(false).
		SetDefaultReadPreference(readpref.Primary()).
		SetDefaultReadConcern(readconcern.Snapshot()).
		SetDefaultWriteConcern(writeconcern.New(writeconcern.WMajority()))

	session, err := ms.d().Client().StartSession(opts)
	if err != nil {
		return nil, wrapMongoError(err)
	}

	if withRetries {
		return session.WithTransaction(ctx, func(sessionCtx mongo.SessionContext) (interface{}, error) { return fn(sessionCtx) })
	}

	if err := session.StartTransaction(); err != nil {
		return nil, wrapMongoError(err)
	}

	sessionContext := mongo.NewSessionContext(ctx, session)
	res, err := fn(sessionContext)
	if err != nil {
		if me := sessionContext.AbortTransaction(sessionContext); me != nil {
			return res, wrapMongoError(me)
		}
		return res, err
	}

	return res, wrapMongoError(sessionContext.CommitTransaction(sessionContext))
}
