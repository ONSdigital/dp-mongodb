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
    When I upsert this record with id 1
            """
            {
                "name": "UpsertName1",
                "age": "UpsertAge1"
            }
            """
    Then There are 1 matched, 1 modified, 0 upserted records
    When I filter on all records
    Then I should find these records
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
    When I upsert this record with id 4
            """
            {
                "name": "UpsertName4",
                "age": "TestAge4"
            } 
            """
    Then There are 0 matched, 0 modified, 1 upserted records, with upsert Id of 4
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
    When I upsertById this record with id 1
            """
            {
                "name": "UpsertByIdName1",
                "age": "UpsertByIdAge1"
            }
            """
    Then There are 1 matched, 1 modified, 0 upserted records
    When I filter on all records
    Then I should find these records
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
    When I upsertById this record with id 4
            """
            {
                "name": "UpsertByIdName4",
                "age": "TestByIdAge4"
            } 
            """
    Then There are 0 matched, 0 modified, 1 upserted records, with upsert Id of 4
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
    When I update this record with id 3
            """
            {
                "name": "UpdateName3",
                "age": "UpdateAge3"
            } 
            """
    Then There are 1 matched, 1 modified, 0 upserted records
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
                        "name": "UpdateName3",
                        "age": "UpdateAge3"
                    }
                ]
                """

  Scenario:
    When I update this record with id 4
            """
            {
                "name": "UpdateName4",
                "age": "UpdateAge4"
            } 
            """
    Then There are 0 matched, 0 modified, 0 upserted records
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
                        "name": "Test3",
                        "age": "TestAge3"
                    }
                ]
                """

  Scenario:
    When I updateById this record with id 3
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
    Then There are 1 matched, 1 modified, 0 upserted records
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
                        "name": "UpdateWithIdName3",
                        "age": "UpdateWithIdAge3"
                    }
                ]
                """

  Scenario:
    When I updateById this record with id 4
            """
            {
                "name": "UpdateWithIdName4",
                "age": "UpdateWithIdAge4"
            } 
            """
    Then There are 0 matched, 0 modified, 0 upserted records
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
                        "name": "Test3",
                        "age": "TestAge3"
                    }
                ]
                """

  Scenario:
    Given I deleteById a record with id 2
    Then There are 1 deleted records
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
                    "id": 3,
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario:
    When I deleteById a record with id 4
    Then There are 0 deleted records
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
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario:
    When I delete a record with id 1
    Then There are 1 deleted records
    When I filter on all records
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
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario:
    When I delete a record with id 4
    Then There are 0 deleted records
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
                    "name": "Test3",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario:
    When I delete a record with name like TestName
    Then There are 2 deleted records
    When I filter on all records
    Then I should find these records
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
    When I insert these records
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
    Then This is the inserted records result
            """
                [4, 5]
            """
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
