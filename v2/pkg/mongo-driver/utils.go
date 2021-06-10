package mongo_driver

// returns true if a update operation updated or upserted a record
func HasUpdatedOrUpserted(updateResult *CollectionUpdateResult) bool {
	return (updateResult.ModifiedCount > 0) || (updateResult.UpsertedCount > 0)
}

// returns true if a remove operation has remove at least one record
func HasRemovedRecords(deleteResult *CollectionDeleteResult) bool {
	return deleteResult.DeletedCount > 0
}
