package mongodb

import (
	"context"
)

type Must struct {
	collection *Collection
}

func newMust(collection *Collection) *Must {
	return &Must{collection}
}

// UpdateId modifies records located by a provided Id selector, must modifiy one record
func (m *Must) UpdateId(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.UpdateId(ctx, id, update)
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
	result, err := m.collection.Update(ctx, selector, update)
	if err != nil {
		return nil, err
	}

	if !HasUpdatedOrUpserted(result) {
		return nil, NewErrNoDocumentFoundError("Must update, no document modified", nil)
	}
	return result, nil
}

// Remove deletes records based on the provided selector, must delete at least one record
func (m *Must) Remove(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.Remove(ctx, selector)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, NewErrNoDocumentFoundError("Must remove, no document deleted", nil)
	}

	return result, nil
}

// RemoveId deletes record based on the id selector must delete at least one record
func (m *Must) RemoveId(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.RemoveId(ctx, id)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, NewErrNoDocumentFoundError("Must remove by id, no document deleted", nil)
	}

	return result, nil
}
