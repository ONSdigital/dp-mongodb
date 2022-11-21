Feature: Transactions

  Background:
    Given I have inserted these Records
    """
    [
        {
            "id": 1,
            "name": "first"
        }
    ]
    """

  Scenario:
    When I update the record in a transaction without interference
    Then the records should match
    """
    [
        {
            "id": 1,
            "name": "second"
        }
    ]
    """

  Scenario:
    When I update the record in a transaction with interference
    Then the records should match
    """
    [
        {
            "id": 1,
            "name": "third"
        }
    ]
    """
