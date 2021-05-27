Feature: Find records
    Background:
        Given I have inserted these Records
            """
            [
                {
                    "id": "1",
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": "2",
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": "3",
                    "name": "TestName1",
                    "age": "TestAge1"
                }
            ]
            """
    Scenario: Find all records
        When I Find all the records in the collection
        Then I should receive these records
            """
            [
                {
                    "id": "1",
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": "2",
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": "3",
                    "name": "TestName1",
                    "age": "TestAge1"
                }
            ]
            """
    Scenario: Count all records
        When I count the records in the collection
        Then I will count 3 records
    
    Scenario: Count with Limit of 2
        When I start a find operation
        And I set the limit to 2
        And I count the records in the collection
        Then I will count 2 records
    
    Scenario: Count with a Skip of 2
        When I start a find operation
        And I skip 2 records
        And I count the records in the collection
        Then I will count 1 records

    Scenario: Count with a Limit of 1 and Skip of 1
        When I start a find operation
        And I set the limit to 1
        And I skip 1 records
        And I count the records in the collection
        Then I will count 1 records

