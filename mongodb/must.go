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

// UpdateById modifies records located by a provided Id selector, must modifiy one record
func (m *Must) UpdateById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.UpdateById(ctx, id, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, NewErrNoDocumentFoundError("Must update by Id, no document matched", nil)
	}

	return result, nil
}

// Update modifies records located by a provided selector, must modifiy one record
func (m *Must) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.Update(ctx, selector, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, NewErrNoDocumentFoundError("Must update, no document matched", nil)
	}

	return result, nil
}

// Delete deletes a record based on the provided selector, must delete at least one record
func (m *Must) Delete(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.Delete(ctx, selector)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, NewErrNoDocumentFoundError("Must delete, no document deleted", nil)
	}

	return result, nil
}

// DeleteMany deletes records based on the provided selector, must delete at least one record
func (m *Must) DeleteMany(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.DeleteMany(ctx, selector)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, NewErrNoDocumentFoundError("Must delete, no document deleted", nil)
	}

	return result, nil
}

// DeleteById deletes record based on the id selector must delete at least one record
func (m *Must) DeleteById(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.DeleteById(ctx, id)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, NewErrNoDocumentFoundError("Must delete by id, no document deleted", nil)
	}

	return result, nil
}
