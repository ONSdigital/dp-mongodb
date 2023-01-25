package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type Must struct {
	collection *Collection
}

func newMust(collection *Collection) *Must {
	return &Must{collection}
}

// UpdateById modifies a single document located by the provided id selector.
// If a  document cannot be found, an ErrNoDocumentFound error is returned
// Deprecated: Use UpdateOne instead
func (m *Must) UpdateById(ctx context.Context, id interface{}, update interface{}) (*CollectionUpdateResult, error) {
	return m.UpdateOne(ctx, bson.M{"_id": id}, update)
}

// Update modifies a single document located by the provided selector.
// If a  document cannot be found, an ErrNoDocumentFound error is returned
// Deprecated: Use UpdateOne instead
func (m *Must) Update(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	return m.UpdateOne(ctx, selector, update)
}

// UpdateOne modifies a single document located by the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// The update must be a document containing update operators and cannot be nil or empty.
// If the selector does not match any documents, an ErrNoDocumentFound is returned
// If the selector matches multiple documents, one will be selected from the matched set, updated and a CollectionUpdateResult with a MatchedCount of 1 will be returned.
func (m *Must) UpdateOne(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.UpdateOne(ctx, selector, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, ErrNoDocumentFound
	}

	return result, nil
}

// UpdateMany modifies multiple documents located by the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// The update must be a document containing update operators and cannot be nil or empty.
// If the selector does not match any documents, an ErrNoDocumentFound is returned
func (m *Must) UpdateMany(ctx context.Context, selector interface{}, update interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.UpdateMany(ctx, selector, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, ErrNoDocumentFound
	}

	return result, nil
}

// Replace modifies a single document located by the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// The newDocument cannot be nil or empty.
// If the selector does not match any documents, an ErrNoDocumentFound is returned
// If the selector matches multiple documents, one will be selected from the matched set, updated and a CollectionUpdateResult with a MatchedCount of 1 will be returned.
func (m *Must) Replace(ctx context.Context, selector interface{}, newDocument interface{}) (*CollectionUpdateResult, error) {
	result, err := m.collection.Replace(ctx, selector, newDocument)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount == 0 {
		return nil, ErrNoDocumentFound
	}

	return result, nil
}

// DeleteById deletes a single document located by the provided id selector. If no document is not found
// an ErrNoDocumentFound error is returned
// Deprecated: Use DeleteOne
func (m *Must) DeleteById(ctx context.Context, id interface{}) (*CollectionDeleteResult, error) {
	return m.DeleteOne(ctx, bson.M{"_id": id})
}

// Delete deletes a single document located by the provided selector. If no document is not found
// an ErrNoDocumentFound error is returned
// Deprecated: Use DeleteOne
func (m *Must) Delete(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	return m.DeleteOne(ctx, selector)
}

// DeleteOne deletes a single document located by the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// If the selector does not match any documents, an ErrNoDocumentFound error is returned
// If the selector matches multiple documents, one will be selected from the matched set, deleted and a CollectionDeleteResult with a DeletedCount of 1 will be returned.
func (m *Must) DeleteOne(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.DeleteOne(ctx, selector)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, ErrNoDocumentFound
	}

	return result, nil
}

// DeleteMany deletes multiple documents located by the provided selector.
// The selector must be a document containing query operators and cannot be nil.
// If the selector does not match any documents, an ErrNoDocumentFound error is returned
func (m *Must) DeleteMany(ctx context.Context, selector interface{}) (*CollectionDeleteResult, error) {
	result, err := m.collection.DeleteMany(ctx, selector)
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		return nil, ErrNoDocumentFound
	}

	return result, nil
}
