package mongodb_test

import (
	"net"
	"strings"
	"testing"
	"time"

	mongoDriver "github.com/ONSdigital/dp-mongodb/v2/mongodb"
	"github.com/ONSdigital/log.go/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestConnectionToMongoDB(t *testing.T) {
	var connectionConfig = &mongoDriver.MongoConnectionConfig{
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:        "test",
		Password:        "test",
		ClusterEndpoint: "localhost:27018",
		Database:        "testDb",
		Collection:      "testCollection",
	}

	if err := checkTcpConnection(connectionConfig.ClusterEndpoint); err != nil {
		log.Event(nil, "mongodb instance not available, skip tests", log.ERROR, log.Error(err))
		t.Skip()
	}
	Convey("When connection to mongodb is attempted", t, func() {

		_, err := mongoDriver.Open(connectionConfig)

		Convey("Then no connection error should happen", func() {
			So(err, ShouldBeNil)
		})
	})
}

// Can be tested by forwarding the document db cluster
// dp ssh develop publishing 4 -- -L 27017:<cluster-url>:27017
func TestConnectionToDocumentDB(t *testing.T) {
	connectionConfig := &mongoDriver.MongoConnectionConfig{
		IsSSL:                   true,
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:                      "test",
		Password:                      "test",
		ClusterEndpoint:               "localhost:27017",
		Database:                      "recipes",
		Collection:                    "recipes",
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
	}
	if err := checkTcpConnection(connectionConfig.ClusterEndpoint); err != nil {
		log.Event(nil, "documentdb instance not available, skip tests", log.ERROR, log.Error(err))
		t.Skip()
	}
	Convey("When connection to documentdb is attempted", t, func() {

		_, err := mongoDriver.Open(connectionConfig)

		Convey("Then no connection error should happen", func() {
			So(err, ShouldBeNil)
		})
	})
}

func checkTcpConnection(connectionString string) error {
	address := strings.Split(connectionString, ":")
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(address[0], address[1]), timeout)
	if err != nil {
		return err
	}
	if conn != nil {
		defer conn.Close()
	}
	return nil
}

func TestMongoConnectionConfig_GetConnectionURIWhen(t *testing.T) {
	connectionConfig := &mongoDriver.MongoConnectionConfig{
		IsSSL:                   true,
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:                      "test",
		Password:                      "test",
		ClusterEndpoint:               "localhost:27017",
		Database:                      "recipes",
		Collection:                    "recipes",
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
	}

	Convey("When Credentials Are Present and ssl is true", t, func() {
		So(connectionConfig.GetConnectionURI(true), ShouldEqual, "mongodb://test:test@localhost:27017/recipes?ssl=true&connect=direct&sslInsecure=true")
	})

	Convey("When Credentials Are Present and ssl is false", t, func() {
		So(connectionConfig.GetConnectionURI(false), ShouldEqual, "mongodb://test:test@localhost:27017/recipes")
	})

	Convey("When Credentials Are Not Configured", t, func() {
		updatedConnectionConfig := connectionConfig
		updatedConnectionConfig.Username = ""
		updatedConnectionConfig.Password = ""
		So(updatedConnectionConfig.GetConnectionURI(false), ShouldEqual, "mongodb://localhost:27017/recipes")
	})
}
