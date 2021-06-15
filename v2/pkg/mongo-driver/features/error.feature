Feature: Error Steps
    Scenario:
        When I start a find operation
        Then Itr All should fail with a wrapped error if an incorrect result param is provided

Scenario:
        When I start a find operation
        Then Find Itr All should fail with a wrapped error if an incorrect result param is provided

Scenario:
        When I start a find operation
        Then Find One should fail with an ErrNoDocumentFound error

    Scenario:
        When I start a find operation
        Then I will count 0 records

   