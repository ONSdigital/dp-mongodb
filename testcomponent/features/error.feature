Feature: Error Steps
  Scenario:
    Given I have inserted these Records
              """
              [
                  {
                      "id": 1,
                      "name": "TestName1",
                      "age": "TestAge1"
                  }
              ]
              """
    When I filter on all records
    Then Find should fail with a wrapped error if an incorrect result param is provided

  Scenario:
    When I filter on all records
    Then Find One should fail with an ErrNoDocumentFound error

  Scenario:
    When I filter on all records
    Then I will count 0 records