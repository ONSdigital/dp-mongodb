Feature: Must operation records

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
                    "name": "Test 3",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario:
    When I Must update this record with id 3
            """
            {
                "name": "Test 3",
                "age": "TestAge3"
            }
            """
    Then there are 1 matched, 0 modified, 0 upserted records
    And Must did not return an error

  Scenario:
    When I Must update this record with id 1
            """
            {
                "name": "UpsertName1",
                "age": "UpsertAge1"
            }
            """
    Then there are 1 matched, 1 modified, 0 upserted records
    And Must did not return an error

  Scenario:
    When I Must update this record with id 10
            """
            {
                "name": "UpsertName1",
                "age": "UpsertAge1"
            }
            """
    Then I should receive a ErrNoDocumentFound error

  Scenario:
    When I Must updateById this record with id 1
            """
            {
                "name": "TestName1",
                "age": "TestAge1"
            }
            """
    Then there are 1 matched, 0 modified, 0 upserted records
    And Must did not return an error

  Scenario:
    When I Must updateById this record with id 3
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
    Then there are 1 matched, 1 modified, 0 upserted records
    And Must did not return an error

  Scenario:
    When I Must updateById this record with id 12
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
    Then I should receive a ErrNoDocumentFound error

  Scenario: Replace matching document
    When I Must replace this record with id 3
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
    Then there are 1 matched, 1 modified, 0 upserted records
    And Must did not return an error

  Scenario: Replace non matching document
    When I Must replace this record with id 12
            """
            {
                "name": "UpdateWithIdName3",
                "age": "UpdateWithIdAge3"
            } 
            """
    Then I should receive a ErrNoDocumentFound error

  Scenario:
    When I Must deleteById a record with id 2
    Then there are 1 deleted records

  Scenario:
    When I Must deleteById a record with id 20
    Then I should receive a ErrNoDocumentFound error

  Scenario:
    When I Must delete a record with id 2
    Then there are 1 deleted records

  Scenario:
    When I Must delete a record with id 20
    Then I should receive a ErrNoDocumentFound error

  Scenario:
    When I Must delete records with name like TestName
    Then there are 2 deleted records

  Scenario:
    When I Must delete records with name like Other
    Then I should receive a ErrNoDocumentFound error

