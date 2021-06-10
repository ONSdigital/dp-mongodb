package mongo_driver

import (
	"context"
)

type Must struct {
	collection *Collection
}

func newMust(collection *Collection) *Must {
	return &Must{collection}
}

// Upsert creates or updates records located by a provided selector, must modifiy one record
func (m *Must) Upsert(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.updateRecord(ctx, selector, update, true)

	if err != nil {
		return nil, err
	}

	if !HasUpdatedOrUpserted(result) {
		return nil, NewErrNoDocumentFoundError("Must upsert, no document modified", nil)
	}

	return result, nil
}

// UpsertId creates or updates records located by a provided Id selector, must modifiy one record
func (m *Must) UpsertId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.updateRecord(ctx, id, update, true)

	if err != nil {
		return nil, err
	}

	if !HasUpdatedOrUpserted(result) {
		return nil, NewErrNoDocumentFoundError("Must upsert by id, no document modified", nil)
	}

	return result, nil
}

// UpdateId modifies records located by a provided Id selector, must modifiy one record
func (m *Must) UpdateId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.updateRecord(ctx, id, update, false)

	if err != nil {
		return nil, err
	}

	if !HasUpdatedOrUpserted(result) {
		return nil, NewErrNoDocumentFoundError("Must update by Id, no document modified", nil)
	}

	return result, nil
}

// Update modifies records located by a provided selector, must modifiy one record
func (m *Must) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.updateRecord(ctx, selector, update, false)
	if err != nil {
		return nil, err
	}

	if !HasUpdatedOrUpserted(result) {
		return nil, NewErrNoDocumentFoundError("Must update, no document modified", nil)
	}
	return result, nil
}
