Feature: Find records
    Background:
        Given I have inserted these Records
            """
            [
                {
                    "id": 1,
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": 2,
                    "name": "TestName2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "TestName3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario: Find all records
        When I start a find operation
        Then I should receive these records
            """
            [
                {
                    "id": 1,
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": 2,
                    "name": "TestName2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "TestName3",
                    "age": "TestAge3"
                }
            ]
            """
    
    Scenario: Find all records with Id > 1
        When I find records with Id > 1
        Then I should receive these records
            """
                [
                    {
                        "id": 2,
                        "name": "TestName2",
                        "age": "TestAge2"
                    },
                    {
                        "id": 3,
                        "name": "TestName3",
                        "age": "TestAge3"
                    }
                ]
            """

    Scenario: Find all records with Id > 1 with Limit of 1
        When I find records with Id > 1
        And I set the limit to 1
        Then I should receive these records
            """
                [
                    {
                        "id": 2,
                        "name": "TestName2",
                        "age": "TestAge2"
                    }
                ]
            """
    Scenario: Find all records with Skip = 1
        When I find records with Id > 1
        And I skip 1 records
        Then I should receive these records
            """
                [
                    {
                        "id": 3,
                        "name": "TestName3",
                        "age": "TestAge3"
                    }
                ]
            """

     Scenario: Find all records with Limit = 1 Skip = 1
        When I find records with Id > 0
        And I set the limit to 1
        And I skip 1 records
        Then I should receive these records
            """
                [
                    {
                        "id": 2,
                        "name": "TestName2",
                        "age": "TestAge2"
                    }
                ]
            """

    Scenario: Find all records sorted by Id desc
        When I start a find operation
        And I sort by ID desc
        Then I should receive these records
            """
            [
                {
                    "id": 3,
                    "name": "TestName3",
                    "age": "TestAge3"
                },
                {
                    "id": 2,
                    "name": "TestName2",
                    "age": "TestAge2"
                },
                {
                    "id": 1,
                    "name": "TestName1",
                    "age": "TestAge1"
                }
            ]
            """

    Scenario: Find all records with the name field
        When I start a find operation
        And I select the field "name"
        Then I should receive these records
            """
            [
                {
                    "id": 1,
                    "name": "TestName1"
                },
                {
                    "id": 2,
                    "name": "TestName2" 
                },
                {
                    "id": 3,
                    "name": "TestName3"
                }
            ]
            """

    Scenario: Count all records
        When I start a find operation
        Then I will count 3 records
    
    Scenario: Count with Limit of 2
        When I start a find operation
        And I set the limit to 2
        Then I will count 2 records
    
    Scenario: Count with a Skip of 2
        When I start a find operation
        And I skip 2 records
        Then I will count 1 records

    Scenario: Count with a Limit of 1 and Skip of 1
        When I start a find operation
        And I set the limit to 1
        And I skip 1 records
        Then I will count 1 records

    Scenario: Count with Find Id > 1 
        When I find records with Id > 1
        Then I will count 2 records
    
    Scenario: Count with Find Id > 1 and Skip 1 
        When I find records with Id > 1
        And I skip 1 records
        Then I will count 1 records

    Scenario: Count with Find Id > 1 and Skip 2
        When I find records with Id > 1
        And I skip 2 records
        Then I will count 0 records

     Scenario: Count with Find Id > 1 and Limit 1 
        When I find records with Id > 1
        And I set the limit to 1
        Then I will count 1 records
    
    Scenario: Count with Find Id > 1 and Limit 2
        When I find records with Id > 1
        And I set the limit to 2
        Then I will count 2 records

     Scenario: Count with Find Id > 1 and Limit 2 And Skip 2
        When I find records with Id > 0
        And I set the limit to 1
        And I skip 2 records
        Then I will count 1 records



