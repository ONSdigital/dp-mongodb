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
    When I start a find operation
    Then Find All should fail with a wrapped error if an incorrect result param is provided

  Scenario:
    When I start a find operation
    Then Find One should fail with an ErrNoDocumentFound error

  Scenario:
    When I start a find operation
    Then I will count 0 records