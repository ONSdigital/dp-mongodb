package testcomponent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	componenttest "github.com/ONSdigital/dp-component-test"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type dataModel struct {
	Id   int `bson:"_id,omitempty" json:"id,omitempty"`
	Name string
	Age  string
}

type find struct {
	query   interface{}
	options []mongoDriver.FindOption
}

type MongoV2Component struct {
	database        string
	collection      string
	rawClient       mongo.Client
	testClient      *mongoDriver.MongoConnection
	find            *find
	insertResult    *mongoDriver.CollectionInsertManyResult
	updateResult    *mongoDriver.CollectionUpdateResult
	deleteResult    *mongoDriver.CollectionDeleteResult
	mustErrorResult error
	ErrorFeature    componenttest.ErrorFeature
}

var noop = func() error { return nil }

func (m *MongoV2Component) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^I have inserted these Records$`, m.insertedTheseRecords)
	ctx.Step(`^I should find these records$`, m.shouldReceiveTheseRecords)
	ctx.Step(`^I should find no records, just a total count of (\d+)$`, m.shouldReceiveNoRecords)
	ctx.Step(`^I should find these distinct fields`, m.shouldReceiveTheseDistinctFields)
	ctx.Step(`^I will count (\d+) records$`, m.countRecords)
	ctx.Step(`^I filter on all records$`, m.findRecords)
	ctx.Step(`^I set the limit to (\d+)`, m.setLimit)
	ctx.Step(`^I skip (\d+) records$`, m.setSkip)
	ctx.Step(`^I set the IgnoreZeroLimit option$`, m.setIgnoreZeroLimit)
	ctx.Step(`^I don't set the IgnoreZeroLimit option$`, noop)
	ctx.Step(`^I filter on records with Id > (\d+)$`, m.findWithId)
	ctx.Step(`^FindOne should give me this one record$`, m.findOneRecord)
	ctx.Step(`^I sort by ID desc`, m.sortByIdDesc)
	ctx.Step(`^I select the field "([^"]*)"$`, m.selectField)
	ctx.Step(`^I filter for records with a distinct value for (\w+)$`, m.distinct)
	ctx.Step(`^I upsertById this record with id (\d+)$`, m.upsertRecordById)
	ctx.Step(`^I upsert this record with id (\d+)$`, m.upsertRecord)
	ctx.Step(`^I updateById this record with id (\d+)$`, m.updateRecordById)
	ctx.Step(`^I update this record with id (\d+)$`, m.updateRecord)
	ctx.Step(`^I deleteById a record with id (\d+)$`, m.deleteRecordById)
	ctx.Step(`^I delete a record with id (\d+)$`, m.deleteRecord)
	ctx.Step(`^I delete a record with name like (\w+)$`, m.deleteRecordByName)
	ctx.Step(`^I insert these records$`, m.insertRecords)
	ctx.Step(`^there are (\d+) matched, (\d+) modified, (\d+) upserted records, with upsert Id of (\d+)$`, m.modifiedCountWithid)
	ctx.Step(`^there are (\d+) matched, (\d+) modified, (\d+) upserted records$`, m.modifiedCount)
	ctx.Step(`^there are (\d+) deleted records$`, m.deletedRecords)
	ctx.Step(`^this is the inserted records result$`, m.insertedRecords)
	ctx.Step(`^Find should fail with a wrapped error if an incorrect result param is provided$`, m.testFindAllError)
	ctx.Step(`^FindOne should fail with an ErrNoDocumentFound error$`, m.testFindOneError)
	ctx.Step(`^I should receive a ErrNoDocumentFound error$`, m.testRecieveErrNoDocumentFoundError)
	ctx.Step(`^Must did not return an error$`, m.testMustDidNotReturnError)
	ctx.Step(`^I Must update this record with id (\d+)$`, m.mustUpdateRecord)
	ctx.Step(`^I Must updateById this record with id (\d+)$`, m.mustUpdateId)
	ctx.Step(`^I Must deleteById a record with id (\d+)$`, m.mustDeleteRecordById)
	ctx.Step(`^I Must delete a record with id (\d+)$`, m.mustDeleteRecord)
	ctx.Step(`^I Must delete records with name like (\w+)$`, m.mustDeleteRecordsByName)
	ctx.Step(`^I update records with name "([^"]*)" age to "([^"]*)"$`, m.iUpdateRecordsWithGroupAgeTo)
	ctx.Step(`^the records should match$`, m.theRecordsShouldMatch)
	ctx.Step(`^I update the record in a transaction (without|with) interference$`, m.runTransaction)
	ctx.Step(`^I sort by name with collation$`, m.findAndSortWithCollation)
}

func newMongoV2Component(database string, collection string, rawClient mongo.Client) *MongoV2Component {
	return &MongoV2Component{database, collection, rawClient,
		mongoDriver.NewMongoConnection(&rawClient, database),
		nil, nil, nil, nil, nil, componenttest.ErrorFeature{}}
}

func (m *MongoV2Component) reset() {
	m.find = nil
	m.insertResult = nil
	m.updateResult = nil
	m.deleteResult = nil
	m.mustErrorResult = nil

	m.ErrorFeature = componenttest.ErrorFeature{}
}

func (m *MongoV2Component) insertedTheseRecords(recordsJson *godog.DocString) error {
	foundRecords := make([]dataModel, 0)
	records := make([]dataModel, 0)

	collection := m.rawClient.Database(m.database).Collection(m.collection)

	err := json.Unmarshal([]byte(recordsJson.Content), &records)
	if err != nil {
		return err
	}

	for _, record := range records {
		_, err := collection.InsertOne(context.Background(), record)

		if err != nil {
			return err
		}
	}

	cursor, err := collection.Find(context.Background(), bson.D{})
	if err != nil {
		return err
	}

	err = cursor.All(context.Background(), &foundRecords)
	if err != nil {
		return err
	}

	assert.EqualValues(&m.ErrorFeature, records, foundRecords)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) findRecords() error {
	m.find = &find{query: bson.D{}}

	return nil
}

func (m *MongoV2Component) theRecordsShouldMatch(recordsJson *godog.DocString) error {
	_ = m.findRecords()
	return m.shouldReceiveTheseRecords(recordsJson)
}

func (m *MongoV2Component) shouldReceiveTheseRecords(recordsJson *godog.DocString) error {
	actualRecords := make([]dataModel, 0)

	_, err := m.testClient.Collection(m.collection).Find(context.Background(), m.find.query, &actualRecords, m.find.options...)
	if err != nil {
		return err
	}

	expectedRecords := make([]dataModel, 0)
	err = json.Unmarshal([]byte(recordsJson.Content), &expectedRecords)
	if err != nil {
		return err
	}

	assert.Equal(&m.ErrorFeature, expectedRecords, actualRecords)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) shouldReceiveNoRecords(totalCount int) error {
	var actualRecords []dataModel

	tc, err := m.testClient.Collection(m.collection).Find(context.Background(), m.find.query, &actualRecords, m.find.options...)
	if err != nil {
		return err
	}

	assert.Equal(&m.ErrorFeature, totalCount, tc)
	assert.Nil(&m.ErrorFeature, actualRecords)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) shouldReceiveTheseDistinctFields(recordsJson *godog.DocString) error {

	var expectedFields []interface{}
	err := json.Unmarshal([]byte(recordsJson.Content), &expectedFields)
	if err != nil {
		return err
	}

	actualFields, err := m.testClient.Collection(m.collection).Distinct(context.Background(), m.find.query.(string), bson.D{})
	if err != nil {
		return err
	}

	assert.EqualValues(&m.ErrorFeature, expectedFields, actualFields)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) countRecords(expected int) error {
	actual, err := m.testClient.Collection(m.collection).Count(context.Background(), m.find.query, m.find.options...)
	if err != nil {
		return err
	}

	assert.Equal(&m.ErrorFeature, expected, actual)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) setLimit(limit int) error {
	m.find.options = append(m.find.options, mongoDriver.Limit(limit))

	return nil
}

func (m *MongoV2Component) setSkip(skip int) error {
	m.find.options = append(m.find.options, mongoDriver.Offset(skip))

	return nil
}

func (m *MongoV2Component) setIgnoreZeroLimit() error {
	m.find.options = append(m.find.options, mongoDriver.IgnoreZeroLimit())

	return nil
}

func (m *MongoV2Component) findWithId(id int) error {
	m.find = &find{query: bson.M{"_id": bson.M{"$gt": id}}}

	return nil
}

func (m *MongoV2Component) sortByIdDesc() error {
	m.find.options = append(m.find.options, mongoDriver.Sort(bson.D{{Key: "_id", Value: -1}}))

	return nil
}

func (m *MongoV2Component) selectField(field string) error {
	m.find.options = append(m.find.options, mongoDriver.Projection(bson.M{field: 1}))

	return nil
}

func (m *MongoV2Component) distinct(field string) error {
	m.find = &find{query: field}

	return nil
}

func (m *MongoV2Component) findOneRecord(recordAsString *godog.DocString) error {
	actualRecord := new(dataModel)
	expectedRecord := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), expectedRecord)
	if err != nil {
		return err
	}

	err = m.testClient.Collection(m.collection).FindOne(context.Background(), m.find.query, &actualRecord, m.find.options...)
	if err != nil {
		return err
	}

	assert.Equal(&m.ErrorFeature, expectedRecord, actualRecord)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) upsertRecordById(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	upsert := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.Collection(m.collection).UpsertOne(context.Background(), bson.M{"_id": id}, upsert)

	return err
}

func (m *MongoV2Component) upsertRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	upsert := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.Collection(m.collection).UpsertOne(context.Background(), bson.M{"_id": id}, upsert)

	return err
}

func (m *MongoV2Component) updateRecordById(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.Collection(m.collection).UpdateOne(context.Background(), bson.M{"_id": id}, update)

	return err
}

func (m *MongoV2Component) updateRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.Collection(m.collection).UpdateOne(context.Background(), bson.M{"_id": id}, update)

	return err
}

func (m *MongoV2Component) iUpdateRecordsWithGroupAgeTo(name, age string) error {
	query := bson.M{"name": name}
	update := bson.D{{
		"$set", bson.D{
			{"age", age},
		}}}

	var err error

	m.updateResult, err = m.testClient.Collection(m.collection).UpdateMany(context.Background(), query, update)

	return err
}

func (m *MongoV2Component) deleteRecordById(id int) error {
	var err error

	m.deleteResult, err = m.testClient.Collection(m.collection).DeleteOne(context.Background(), bson.M{"_id": id})

	return err
}

func (m *MongoV2Component) deleteRecord(id int) error {
	var err error

	m.deleteResult, err = m.testClient.Collection(m.collection).DeleteOne(context.Background(), bson.M{"_id": id})

	return err
}

func (m *MongoV2Component) deleteRecordByName(name string) error {
	var err error

	selector := bson.D{{Key: "name", Value: primitive.Regex{Pattern: ".*" + name + ".*"}}}

	m.deleteResult, err = m.testClient.Collection(m.collection).DeleteMany(context.Background(), selector)

	return err
}

func (m *MongoV2Component) modifiedCountWithid(matched, modified, upserted, upsertId int) error {
	assert.Equal(&m.ErrorFeature, matched, m.updateResult.MatchedCount)
	assert.Equal(&m.ErrorFeature, modified, m.updateResult.ModifiedCount)
	assert.Equal(&m.ErrorFeature, upserted, m.updateResult.UpsertedCount)
	assert.EqualValues(&m.ErrorFeature, upsertId, m.updateResult.UpsertedID)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) modifiedCount(matched, modified, upserted int) error {
	assert.Equal(&m.ErrorFeature, matched, m.updateResult.MatchedCount, "Matched Count")
	assert.Equal(&m.ErrorFeature, modified, m.updateResult.ModifiedCount, "Modified Count")
	assert.Equal(&m.ErrorFeature, upserted, m.updateResult.UpsertedCount, "Upsert Count")
	assert.Empty(&m.ErrorFeature, m.updateResult.UpsertedID)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) deletedRecords(deleted int) error {
	assert.Equal(&m.ErrorFeature, deleted, m.deleteResult.DeletedCount)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) insertRecords(recordsJson *godog.DocString) error {
	records := make([]dataModel, 0)

	err := json.Unmarshal([]byte(recordsJson.Content), &records)
	if err != nil {
		return err
	}

	testRecords := []interface{}{records[0], records[1]}

	m.insertResult, err = m.testClient.Collection(m.collection).InsertMany(context.Background(), testRecords)
	if err != nil {
		return err
	}

	return err
}

func (m *MongoV2Component) insertedRecords(recordsJson *godog.DocString) error {
	expected := make([]int32, 0)
	actual := make([]int32, 0)

	err := json.Unmarshal([]byte(recordsJson.Content), &expected)
	if err != nil {
		return err
	}

	for _, element := range m.insertResult.InsertedIds {
		actual = append(actual, element.(int32))
	}

	assert.Equal(&m.ErrorFeature, expected, actual)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testFindAllError() error {
	badResult := 1

	_, err := m.testClient.Collection(m.collection).Find(context.Background(), m.find.query, &badResult, m.find.options...)

	assert.True(&m.ErrorFeature, mongoDriver.IsServerErr(err))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testFindOneError() error {
	var result dataModel

	err := m.testClient.Collection(m.collection).FindOne(context.Background(), m.find.query, &result, m.find.options...)

	assert.True(&m.ErrorFeature, errors.Is(err, mongoDriver.ErrNoDocumentFound))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) mustUpdateId(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	update := bson.M{"$set": record}
	m.updateResult, m.mustErrorResult = m.testClient.Collection(m.collection).Must().UpdateOne(context.Background(), bson.M{"_id": id}, update)

	return nil
}

func (m *MongoV2Component) mustUpdateRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	m.updateResult, m.mustErrorResult = m.testClient.Collection(m.collection).Must().UpdateOne(context.Background(), bson.M{"_id": id}, bson.M{"$set": record})

	return nil
}

func (m *MongoV2Component) testRecieveErrNoDocumentFoundError() error {
	assert.True(&m.ErrorFeature, errors.Is(m.mustErrorResult, mongoDriver.ErrNoDocumentFound))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testMustDidNotReturnError() error {
	assert.NoError(&m.ErrorFeature, m.mustErrorResult)
	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) mustDeleteRecordById(id int) error {
	m.deleteResult, m.mustErrorResult = m.testClient.Collection(m.collection).Must().DeleteOne(context.Background(), bson.M{"_id": id})

	return nil
}

func (m *MongoV2Component) mustDeleteRecord(id int) error {
	m.deleteResult, m.mustErrorResult = m.testClient.Collection(m.collection).Must().DeleteOne(context.Background(), bson.M{"_id": id})

	return nil
}

func (m *MongoV2Component) mustDeleteRecordsByName(name string) error {
	selector := bson.D{{Key: "name", Value: primitive.Regex{Pattern: ".*" + name + ".*"}}}

	m.deleteResult, m.mustErrorResult = m.testClient.Collection(m.collection).Must().DeleteMany(context.Background(), selector)

	return nil
}

func (m *MongoV2Component) runTransaction(interference string) error {
	_, e := m.testClient.RunTransaction(context.Background(), false, func(transactionCtx context.Context) (interface{}, error) {
		var obj dataModel
		err := m.testClient.Collection(m.collection).FindOne(transactionCtx, bson.M{"_id": 1}, &obj)
		if err != nil {
			return nil, fmt.Errorf("could not find object in collection (%s): %w", m.collection, err)
		}

		if interference == "with" {
			_, err = m.testClient.Collection(m.collection).UpdateOne(context.Background(), bson.M{"_id": 1}, bson.M{"$set": bson.M{"name": "third"}})
			if err != nil {
				return nil, fmt.Errorf("interleave write failed in collection (%s): %w", m.collection, err)
			}
		}

		if obj.Name == "first" {
			obj.Name = "second"
			_, err = m.testClient.Collection(m.collection).UpdateOne(transactionCtx, bson.M{"_id": 1}, bson.M{"$set": obj})
			if err != nil {
				return nil, fmt.Errorf("could not write object in collection (%s): %w", m.collection, err)
			}
		}

		return obj, nil
	})

	if interference == "with" {
		assert.Error(&m.ErrorFeature, e)
	} else {
		assert.Nil(&m.ErrorFeature, e)
	}

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) findAndSortWithCollation() error {
	m.find.options = append(m.find.options, mongoDriver.Sort(bson.D{{Key: "name", Value: 1}}), mongoDriver.Collation(&options.Collation{Locale: "en", Strength: 2}))

	return nil
}
