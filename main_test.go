package mongodb_test

import (
	"flag"
	"os"
	"testing"

	componentTest "github.com/ONSdigital/dp-component-test"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

const MongoVersion = "4.4.8"
const MongoPort = 27017
const DatabaseName = "testing"
const CollectionName = "testCollection"

type MongoDBComponentTest struct {
	MongoFeature     *componentTest.MongoFeature
	MongoV2Component *MongoV2Component
}

var componentFlag = flag.Bool("component", false, "perform component tests")

func (m *MongoDBComponentTest) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		m.MongoFeature = componentTest.NewMongoFeature(componentTest.MongoOptions{MongoVersion: MongoVersion, DatabaseName: DatabaseName})
		m.MongoV2Component = newMongoV2Component(DatabaseName, CollectionName, m.MongoFeature.Client)
	})
	ctx.AfterSuite(func() {
		m.MongoFeature.Close()
	})
}

func (m *MongoDBComponentTest) InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*godog.Scenario) {
		m.MongoFeature.Reset()
		m.MongoV2Component.reset()
	})

	m.MongoFeature.RegisterSteps(ctx)
	m.MongoV2Component.RegisterSteps(ctx)
}

func TestMain(m *testing.M) {
	var status int

	flag.Parse()
	if *componentFlag {
		var opts = godog.Options{
			Output: colors.Colored(os.Stdout),
			Paths:  flag.Args(),
			Format: "pretty",
		}

		mongoDBTest := &MongoDBComponentTest{}

		status = godog.TestSuite{
			Name:                 "component_tests",
			TestSuiteInitializer: mongoDBTest.InitializeTestSuite,
			ScenarioInitializer:  mongoDBTest.InitializeScenario,
			Options:              &opts,
		}.Run()
	}

	os.Exit(status)
}
