Feature: Find One record
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
                },
                {
                    "id": 4,
                    "name": "TestName4",
                    "age": "TestAge4"
                }
            ]
            """
    
    Scenario: Find one record by Id
        When I find records with Id > 1
        Then I find this one record
            """
            {
                "id": 2,
                "name": "TestName2" ,
                "age": "TestAge2"
            }
            """
    
    Scenario: Find one record by Id and skip 1
        When I find records with Id > 1
        And I skip 1 records
        Then I find this one record
            """
            {
                "id": 3,
                "name": "TestName3" ,
                "age": "TestAge3"
            }
            """

    Scenario: Find one record by Id and sort id desc
        When I find records with Id > 1
        And I sort by ID desc
        Then I find this one record
            """
            {
                "id": 4,
                "name": "TestName4" ,
                "age": "TestAge4"
            }
            """

    Scenario: Find one field select
        When I find records with Id > 2
        And I select the field "name"
        Then I find this one record
            """
            {
                "id": 3,
                "name": "TestName3" 
            }
            """

        
        