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
                    "age": "TestAge2"
                }
            ]
        """
    Scenario: Find all records
        When I filter on all records
        Then I should find these records
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
                    "age": "TestAge2"
                }
            ]
            """

    Scenario: Find all records with Id > 1
        When I filter on records with Id > 1
        Then I should find these records
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
                        "age": "TestAge2"
                    }
                ]
            """

    Scenario: Find all records with Id > 1 with Limit of 1
        When I filter on records with Id > 1
        And I set the limit to 1
        Then I should find these records
            """
                [
                    {
                        "id": 2,
                        "name": "TestName2",
                        "age": "TestAge2"
                    }
                ]
            """

    Scenario: Find all records with Id > 1 with Limit of 0
        When I filter on records with Id > 1
        And I set the limit to 0
        And I don't set the IgnoreZeroLimit option
        Then I should find no records, just a total count of 2

    Scenario: Find all records with Id > 1 with Limit of 0, but ignore limit
        When I filter on records with Id > 1
        And I set the limit to 0
        But I set the IgnoreZeroLimit option
        Then I should find these records
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
                        "age": "TestAge2"
                    }
                ]
            """

    Scenario: Find all records with Skip = 1
        When I filter on records with Id > 1
        And I skip 1 records
        Then I should find these records
            """
                [
                    {
                        "id": 3,
                        "name": "TestName3",
                        "age": "TestAge2"
                    }
                ]
            """

     Scenario: Find all records with Limit = 1 Skip = 1
        When I filter on records with Id > 0
        And I set the limit to 1
        And I skip 1 records
        Then I should find these records
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
        When I filter on all records
        And I sort by ID desc
        Then I should find these records
            """
            [
                {
                    "id": 3,
                    "name": "TestName3",
                    "age": "TestAge2"
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
        When I filter on all records
        And I select the field "name"
        Then I should find these records
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

    Scenario: Find all records sorted using collation
        When I insert these records
            """
            [
                {
                    "id": 4,
                    "name": "b_name_4",
                    "age": "b_age_4"
                },
                {
                    "id": 5,
                    "name": "a_name_5",
                    "age": "a_age_5"
                }
            ]
            """
        And I filter on all records
        And I sort by name with collation
        Then I should find these records
            """
            [
                {
                    "id": 5,
                    "name": "a_name_5",
                    "age": "a_age_5"
                },
                {
                    "id": 4,
                    "name": "b_name_4",
                    "age": "b_age_4"
                },
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
                    "age": "TestAge2"
                }
            ]
            """

    Scenario: Count all records
        When I filter on all records
        Then I will count 3 records

    Scenario: Count with Limit of 0 (limit will be ignored)
        When I filter on all records
        And I set the limit to 0
        Then I will count 3 records

    Scenario: Count with Limit of 2
        When I filter on all records
        And I set the limit to 2
        Then I will count 2 records
    
    Scenario: Count with a Skip of 2
        When I filter on all records
        And I skip 2 records
        Then I will count 1 records

    Scenario: Count with a Limit of 1 and Skip of 1
        When I filter on all records
        And I set the limit to 1
        And I skip 1 records
        Then I will count 1 records

    Scenario: Count with Find Id > 1 
        When I filter on records with Id > 1
        Then I will count 2 records
    
    Scenario: Count with Find Id > 1 and Skip 1 
        When I filter on records with Id > 1
        And I skip 1 records
        Then I will count 1 records

    Scenario: Count with Find Id > 1 and Skip 2
        When I filter on records with Id > 1
        And I skip 2 records
        Then I will count 0 records

     Scenario: Count with Find Id > 1 and Limit 1 
        When I filter on records with Id > 1
        And I set the limit to 1
        Then I will count 1 records
    
    Scenario: Count with Find Id > 1 and Limit 2
        When I filter on records with Id > 1
        And I set the limit to 2
        Then I will count 2 records

     Scenario: Count with Find Id > 1 and Limit 2 And Skip 2
        When I filter on records with Id > 0
        And I set the limit to 1
        And I skip 2 records
        Then I will count 1 records

    Scenario: Find distinct records
        When I filter for records with a distinct value for age
        Then I should find these distinct fields
        """
            ["TestAge1", "TestAge2"]
        """