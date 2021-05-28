Feature: Collection records
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
    Scenario:
        Given I upsert this record with id 1
            """
            {
                "name": "UpsertName1",
                "age": "UpsertAge1"
            }
            """
        When I Find all the records in the collection
        Then I should receive these records
         """
            [
                {
                    "id": 1,
                    "name": "UpsertName1",
                    "age": "UpsertAge1"
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
    Scenario:
        Given I upsert this record with id 4
            """
            {
                "name": "UpsertName4",
                "age": "TestAge4"
            }
            """
        When I Find all the records in the collection
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
                },
                {
                    "id": 4,
                    "name": "UpsertName4",
                    "age": "TestAge4"
                }
            ]
            """