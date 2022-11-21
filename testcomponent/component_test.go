package testcomponent

import (
	"flag"
	"os"
	"testing"

	componentTest "github.com/ONSdigital/dp-component-test"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

const (
	MongoVersion   = "4.4.8"
	DatabaseName   = "testing"
	ReplicaSetName = "test-replica-set"
	CollectionName = "testCollection"
)

type MongoDBComponentTest struct {
	MongoFeature     *componentTest.MongoFeature
	MongoV2Component *MongoV2Component
}

var componentFlag = flag.Bool("component", false, "perform component tests")

func (m *MongoDBComponentTest) InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		m.MongoFeature = componentTest.NewMongoFeature(componentTest.MongoOptions{MongoVersion: MongoVersion, DatabaseName: DatabaseName, ReplicaSetName: ReplicaSetName})
		m.MongoV2Component = newMongoV2Component(DatabaseName, CollectionName, m.MongoFeature.Client)
	})
	ctx.AfterSuite(func() {
		_ = m.MongoFeature.Close()
	})
}

func (m *MongoDBComponentTest) InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*godog.Scenario) {
		_ = m.MongoFeature.Reset()
		m.MongoV2Component.reset()
	})

	m.MongoFeature.RegisterSteps(ctx)
	m.MongoV2Component.RegisterSteps(ctx)
}

func TestComponent(t *testing.T) {
	if *componentFlag {
		var opts = godog.Options{
			Output: colors.Colored(os.Stdout),
			Paths:  flag.Args(),
			Format: "pretty",
		}

		mongoDBTest := &MongoDBComponentTest{}

		status := godog.TestSuite{
			Name:                 "component_tests",
			TestSuiteInitializer: mongoDBTest.InitializeTestSuite,
			ScenarioInitializer:  mongoDBTest.InitializeScenario,
			Options:              &opts,
		}.Run()

		if status > 0 {
			t.Errorf("component testing from godog test suite failed with status %d", status)
		}

	} else {
		t.Skip("skipping component testing")
	}
}
