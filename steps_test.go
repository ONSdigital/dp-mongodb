package mongodb_test

import (
	"context"
	"encoding/json"

	componenttest "github.com/ONSdigital/dp-component-test"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type dataModel struct {
	Id   int `bson:"_id" json:"id"`
	Name string
	Age  string
}

type MongoV2Component struct {
	database        string
	collection      string
	rawClient       mongo.Client
	testClient      *mongoDriver.MongoConnection
	find            *mongoDriver.Find
	insertResult    *mongoDriver.CollectionInsertManyResult
	updateResult    *mongoDriver.CollectionUpdateResult
	deleteResult    *mongoDriver.CollectionDeleteResult
	mustErrorResult error
	ErrorFeature    componenttest.ErrorFeature
}

func (m *MongoV2Component) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^I have inserted these Records$`, m.insertedTheseRecords)
	ctx.Step(`^I should receive these records$`, m.shouldReceiveTheseRecords)
	ctx.Step(`^I will count (\d+) records$`, m.countRecords)
	ctx.Step(`^I start a find operation`, m.findRecords)
	ctx.Step(`^I set the limit to (\d+)`, m.setLimit)
	ctx.Step(`^I skip (\d+) records$`, m.setSkip)
	ctx.Step(`^I find records with Id > (\d+)$`, m.findWithId)
	ctx.Step(`^I find this one record$`, m.findOneRecord)
	ctx.Step(`^I sort by ID desc`, m.sortByIdDesc)
	ctx.Step(`^I select the field "([^"]*)"$`, m.selectField)
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
	ctx.Step(`^Itr All should fail with a wrapped error if an incorrect result param is provided$`, m.testErrorItrAll)
	ctx.Step(`^Find Itr All should fail with a wrapped error if an incorrect result param is provided$`, m.testFindErrorItrAll)
	ctx.Step(`^Find One should fail with an ErrNoDocumentFound error$`, m.testFindOneError)
	ctx.Step(`^I should receive a ErrNoDocumentFound error$`, m.testRecieveErrNoDocumentFoundError)
	ctx.Step(`^Must did not return an error$`, m.testMustDidNotReturnError)
	ctx.Step(`^I Must update this record with id (\d+)$`, m.mustUpdateRecord)
	ctx.Step(`^I Must updateById this record with id (\d+)$`, m.mustUpdateId)
	ctx.Step(`^I Must deleteById a record with id (\d+)$`, m.mustDeleteRecordById)
	ctx.Step(`^I Must delete a record with id (\d+)$`, m.mustDeleteRecord)
	ctx.Step(`^I Must delete records with name like (\w+)$`, m.mustDeleteRecordsByName)
}

func newMongoV2Component(database string, collection string, rawClient mongo.Client) *MongoV2Component {
	return &MongoV2Component{database, collection, rawClient,
		mongoDriver.NewMongoConnection(&rawClient, database, collection),
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

	assert.ElementsMatch(&m.ErrorFeature, records, foundRecords)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) findRecords() error {
	m.find = m.testClient.C(m.collection).Find(bson.D{})

	return nil
}

func (m *MongoV2Component) shouldReceiveTheseRecords(recordsJson *godog.DocString) error {
	actualRecords := make([]dataModel, 0)

	err := m.find.Iter().All(context.Background(), &actualRecords)
	if err != nil {
		return err
	}

	expectedRecords := make([]dataModel, 0)

	err = json.Unmarshal([]byte(recordsJson.Content), &expectedRecords)
	if err != nil {
		return err
	}

	assert.ElementsMatch(&m.ErrorFeature, expectedRecords, actualRecords)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) countRecords(expected int) error {
	actual, err := m.find.Count(context.Background())
	if err != nil {
		return err
	}

	assert.EqualValues(&m.ErrorFeature, int(expected), int(actual))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) setLimit(limit int) error {
	m.find.Limit(limit)

	return nil
}

func (m *MongoV2Component) setSkip(skip int) error {
	m.find.Skip(skip)

	return nil
}

func (m *MongoV2Component) findWithId(id int) error {
	m.find = m.testClient.C(m.collection).Find(bson.M{"_id": bson.M{"$gt": id}})

	return nil
}

func (m *MongoV2Component) sortByIdDesc() error {
	m.find.Sort(bson.D{{Key: "_id", Value: -1}})

	return nil
}

func (m *MongoV2Component) selectField(field string) error {
	m.find.Select(bson.M{field: 1})

	return nil
}

func (m *MongoV2Component) findOneRecord(recordAsString *godog.DocString) error {
	actualRecord := new(dataModel)
	expectedRecord := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), expectedRecord)
	if err != nil {
		return err
	}

	err = m.find.One(context.Background(), &actualRecord)
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

	m.updateResult, err = m.testClient.C(m.collection).UpsertById(context.Background(), id, upsert)

	return err
}

func (m *MongoV2Component) upsertRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	idQuery := bson.D{{Key: "_id", Value: id}}

	upsert := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.C(m.collection).Upsert(context.Background(), idQuery, upsert)

	return err
}

func (m *MongoV2Component) updateRecordById(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.C(m.collection).UpdateById(context.Background(), id, update)

	return err
}

func (m *MongoV2Component) updateRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	idQuery := bson.D{{Key: "_id", Value: id}}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, err = m.testClient.C(m.collection).Update(context.Background(), idQuery, update)

	return err
}

func (m *MongoV2Component) deleteRecordById(id int) error {
	var err error

	m.deleteResult, err = m.testClient.C(m.collection).DeleteById(context.Background(), id)

	return err
}

func (m *MongoV2Component) deleteRecord(id int) error {
	var err error

	idQuery := bson.D{{Key: "_id", Value: id}}

	m.deleteResult, err = m.testClient.C(m.collection).Delete(context.Background(), idQuery)

	return err
}

func (m *MongoV2Component) deleteRecordByName(name string) error {
	var err error

	selector := bson.D{{Key: "name", Value: primitive.Regex{Pattern: ".*" + name + ".*"}}}

	m.deleteResult, err = m.testClient.C(m.collection).DeleteMany(context.Background(), selector)

	return err
}

func (m *MongoV2Component) modifiedCountWithid(matched, modified, upserted, upsertId int) error {
	assert.Equal(&m.ErrorFeature, matched, m.updateResult.MatchedCount)
	assert.Equal(&m.ErrorFeature, modified, m.updateResult.ModifiedCount)
	assert.Equal(&m.ErrorFeature, upserted, m.updateResult.UpsertedCount)
	assert.EqualValues(&m.ErrorFeature, upsertId, m.updateResult.UpsertedID)

	if modified != 0 || upserted != 0 {
		assert.True(&m.ErrorFeature, mongoDriver.HasUpdatedOrUpserted(m.updateResult))
	} else {
		assert.False(&m.ErrorFeature, mongoDriver.HasUpdatedOrUpserted(m.updateResult))
	}

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) modifiedCount(matched, modified, upserted int) error {
	assert.Equal(&m.ErrorFeature, matched, m.updateResult.MatchedCount)
	assert.Equal(&m.ErrorFeature, modified, m.updateResult.ModifiedCount)
	assert.Equal(&m.ErrorFeature, upserted, m.updateResult.UpsertedCount)
	assert.Empty(&m.ErrorFeature, m.updateResult.UpsertedID)

	if modified != 0 || upserted != 0 {
		assert.True(&m.ErrorFeature, mongoDriver.HasUpdatedOrUpserted(m.updateResult))
	} else {
		assert.False(&m.ErrorFeature, mongoDriver.HasUpdatedOrUpserted(m.updateResult))
	}

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

	m.insertResult, err = m.testClient.C(m.collection).InsertMany(context.Background(), testRecords)
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

	assert.ElementsMatch(&m.ErrorFeature, expected, actual)

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testErrorItrAll() error {
	badResult := 1

	err := m.find.Iter().All(context.Background(), &badResult)

	assert.True(&m.ErrorFeature, mongoDriver.IsServerErr(err))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testFindErrorItrAll() error {
	badResult := 1

	err := m.find.IterAll(context.Background(), &badResult)

	assert.True(&m.ErrorFeature, mongoDriver.IsServerErr(err))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testFindOneError() error {
	var result dataModel

	err := m.find.One(context.Background(), &result)

	assert.True(&m.ErrorFeature, mongoDriver.IsErrNoDocumentFound(err))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) mustUpdateId(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, m.mustErrorResult = m.testClient.C(m.collection).Must().UpdateById(context.Background(), id, update)

	return nil
}

func (m *MongoV2Component) mustUpdateRecord(id int, recordAsString *godog.DocString) error {
	record := new(dataModel)

	err := json.Unmarshal([]byte(recordAsString.Content), &record)
	if err != nil {
		return err
	}

	idQuery := bson.D{{Key: "_id", Value: id}}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: record.Name}, {Key: "age", Value: record.Age}}}}

	m.updateResult, m.mustErrorResult = m.testClient.C(m.collection).Must().Update(context.Background(), idQuery, update)

	return nil
}

func (m *MongoV2Component) testRecieveErrNoDocumentFoundError() error {
	assert.True(&m.ErrorFeature, mongoDriver.IsErrNoDocumentFound(m.mustErrorResult))

	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) testMustDidNotReturnError() error {
	assert.NoError(&m.ErrorFeature, m.mustErrorResult)
	return m.ErrorFeature.StepError()
}

func (m *MongoV2Component) mustDeleteRecordById(id int) error {
	m.deleteResult, m.mustErrorResult = m.testClient.C(m.collection).Must().DeleteById(context.Background(), id)

	return nil
}

func (m *MongoV2Component) mustDeleteRecord(id int) error {
	idQuery := bson.D{{Key: "_id", Value: id}}

	m.deleteResult, m.mustErrorResult = m.testClient.C(m.collection).Must().Delete(context.Background(), idQuery)

	return nil
}

func (m *MongoV2Component) mustDeleteRecordsByName(name string) error {
	selector := bson.D{{Key: "name", Value: primitive.Regex{Pattern: ".*" + name + ".*"}}}

	m.deleteResult, m.mustErrorResult = m.testClient.C(m.collection).Must().DeleteMany(context.Background(), selector)

	return nil
}
