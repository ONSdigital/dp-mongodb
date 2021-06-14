Feature: Error Steps
    Scenario:
        When I Find all the records in the collection
        Then Itr All should fail with a wrapped error if an incorrect result param is provided

Scenario:
        When I Find all the records in the collection
        Then Find Itr All should fail with a wrapped error if an incorrect result param is provided

Scenario:
        When I start a find operation
        Then Find One should fail with an ErrCollectionNotFound error

   