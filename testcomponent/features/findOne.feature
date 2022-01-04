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
        When I filter on records with Id > 1
        Then find one should give me this one record
            """
            {
                "id": 2,
                "name": "TestName2" ,
                "age": "TestAge2"
            }
            """
    
    Scenario: Find one record by Id and skip 1
        When I filter on records with Id > 1
        And I skip 1 records
        Then find one should give me this one record
            """
            {
                "id": 3,
                "name": "TestName3" ,
                "age": "TestAge3"
            }
            """

    Scenario: Find one record by Id and sort id desc
        When I filter on records with Id > 1
        And I sort by ID desc
        Then find one should give me this one record
            """
            {
                "id": 4,
                "name": "TestName4" ,
                "age": "TestAge4"
            }
            """

    Scenario: Find one field select
        When I filter on records with Id > 2
        And I select the field "name"
        Then find one should give me this one record
            """
            {
                "id": 3,
                "name": "TestName3" 
            }
            """

        
        