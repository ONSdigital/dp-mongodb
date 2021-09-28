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
                    "name": "Test3",
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
        When I start a find operation
        Then there are 1 matched, 1 modified, 0 upserted records
        And I should receive these records
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
                    "name": "Test3",
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
        When I start a find operation
        Then there are 0 matched, 0 modified, 1 upserted records, with upsert Id of 4
        And I should receive these records
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
                    "name": "Test3",
                    "age": "TestAge3"
                },
                {
                    "id": 4,
                    "name": "UpsertName4",
                    "age": "TestAge4"
                }
            ]
            """
    Scenario:
        Given I upsertById this record with id 1
            """
            {
                "name": "UpsertByIdName1",
                "age": "UpsertByIdAge1"
            }
            """
        When I start a find operation
        Then there are 1 matched, 1 modified, 0 upserted records
        And I should receive these records
         """
            [
                {
                    "id": 1,
                    "name": "UpsertByIdName1",
                    "age": "UpsertByIdAge1"
                },
                {
                    "id": 2,
                    "name": "TestName2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I upsertById this record with id 4
            """
            {
                "name": "UpsertByIdName4",
                "age": "TestByIdAge4"
            } 
            """
        When I start a find operation
        Then there are 0 matched, 0 modified, 1 upserted records, with upsert Id of 4
        And I should receive these records
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
                    "name": "Test3",
                    "age": "TestAge3"
                },
                {
                    "id": 4,
                    "name": "UpsertByIdName4",
                    "age": "TestByIdAge4"
                }
            ]
            """
    Scenario:
        Given I update this record with id 3
            """
            {
                "name": "UpdateName3",
                "age": "UpdateAge3"
            } 
            """
            When I start a find operation
            Then there are 1 matched, 1 modified, 0 upserted records
            And I should receive these records
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
                        "name": "UpdateName3",
                        "age": "UpdateAge3"
                    }
                ]
                """
     Scenario:
        Given I update this record with id 4
            """
            {
                "name": "UpdateName4",
                "age": "UpdateAge4"
            } 
            """
            When I start a find operation
            Then there are 0 matched, 0 modified, 0 upserted records
            And I should receive these records
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
                        "name": "Test3",
                        "age": "TestAge3"
                    }
                ]
                """
        Scenario:
        Given I updateById this record with id 3
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
            When I start a find operation
            Then there are 1 matched, 1 modified, 0 upserted records
            And I should receive these records
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
                        "name": "UpdateWithIdName3",
                        "age": "UpdateWithIdAge3"
                    }
                ]
                """
     Scenario:
        Given I updateById this record with id 4
            """
            {
                "name": "UpdateWithIdName4",
                "age": "UpdateWithIdAge4"
            } 
            """
            When I start a find operation
            Then there are 0 matched, 0 modified, 0 upserted records
            And I should receive these records
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
                        "name": "Test3",
                        "age": "TestAge3"
                    }
                ]
                """
     Scenario:
        Given I deleteById a record with id 2
            When I start a find operation
            Then there are 1 deleted records
            And I should receive these records
            """
            [
                {
                    "id": 1,
                    "name": "TestName1",
                    "age": "TestAge1"
                },
                {
                    "id": 3,
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I deleteById a record with id 4
            When I start a find operation
            Then there are 0 deleted records
            And I should receive these records
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
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I delete a record with id 1
            When I start a find operation
            Then there are 1 deleted records
            And I should receive these records
            """
            [
                {
                    "id": 2,
                    "name": "TestName2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I delete a record with id 4
            When I start a find operation
            Then there are 0 deleted records
            And I should receive these records
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
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I delete a record with name like TestName
            When I start a find operation
            Then there are 2 deleted records
            And I should receive these records
            """
            [
                {
                    "id": 3,
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """
    Scenario:
        Given I insert these records
            """
            [
                {
                    "id": 4,
                    "name": "TestName4",
                    "age": "TestAge4"
                },
                {
                    "id": 5,
                    "name": "TestName5",
                    "age": "TestAge5"
                }
            ]
            """
            When I start a find operation
            Then this is the inserted records result 
            """
                [4, 5]
            """
            And I should receive these records
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
                    "name": "Test3",
                    "age": "TestAge3"
                },
                {
                    "id": 4,
                    "name": "TestName4",
                    "age": "TestAge4"
                },
                {
                    "id": 5,
                    "name": "TestName5",
                    "age": "TestAge5"
                }
            ]
            """
