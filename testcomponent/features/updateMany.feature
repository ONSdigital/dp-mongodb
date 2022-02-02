Feature: Update Many Records

  Background:
    Given I have inserted these Records
            """
            [
                {
                    "id": 1,
                    "name": "Tester 1",
                    "age": "TestAge1"
                },
                {
                    "id": 2,
                    "name": "Tester 2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "Tester 2",
                    "age": "TestAge3"
                }
            ]
            """

  Scenario: Update One Record
    When I update records with name "Tester 1" age to "Testing Age"
    Then there are 1 matched, 1 modified, 0 upserted records
    And the records should match
      """
            [
                {
                    "id": 1,
                    "name": "Tester 1",
                    "age": "Testing Age"
                },
                {
                    "id": 2,
                    "name": "Tester 2",
                    "age": "TestAge2"
                },
                {
                    "id": 3,
                    "name": "Tester 2",
                    "age": "TestAge3"
                }
            ]
       """
  Scenario: Update Many Records
    When I update records with name "Tester 2" age to "Testing Age"
    Then there are 2 matched, 2 modified, 0 upserted records
    And the records should match
      """
            [
                {
                    "id": 1,
                    "name": "Tester 1",
                    "age": "TestAge1"
                },
                {
                    "id": 2,
                    "name": "Tester 2",
                    "age": "Testing Age"
                },
                {
                    "id": 3,
                    "name": "Tester 2",
                    "age": "Testing Age"
                }
            ]
       """

