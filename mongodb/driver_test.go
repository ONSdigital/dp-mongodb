package mongodb_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mim "github.com/ONSdigital/dp-mongodb-in-memory"
	mongoDriver "github.com/ONSdigital/dp-mongodb/v3/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	. "github.com/smartystreets/goconvey/convey"
)

// Example of how to connect to a Mongo DB server with SSL, in this case a connection via ssh port forwarding to a DocumentDB server
// cluster: `dp ssh develop publishing 1 -p 27017:develop-docdb-cluster.cluster-cpviojtnaxsj.eu-west-1.docdb.amazonaws.com:27017`
func ExampleOpen() {
	connectionConfig := &mongoDriver.MongoConnectionConfig{
		ClusterEndpoint: "localhost:27017",
		Database:        "recipes",
		Collection:      "recipes",
		Username:        "XXX- username for recipe-api for authentication",
		Password:        "XXX - the password for the above username",

		ConnectTimeoutInSeconds:       5,
		QueryTimeoutInSeconds:         5,
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
		TLSConnectionConfig: mongoDriver.TLSConnectionConfig{
			IsSSL:              true,
			VerifyCert:         true,
			CACertChain:        "./test/data/rds-combined-ca-bundle.pem",
			RealHostnameForSSH: "develop-docdb-cluster.cluster-cpviojtnaxsj.eu-west-1.docdb.amazonaws.com",
		},
	}

	mongoDB, err := mongoDriver.Open(connectionConfig)
	if err != nil {
		// log error, cannot use mongo db
	}

	// Can now work with the mongo db
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, _ = mongoDB.GetConfiguredCollection().Insert(ctx, bson.M{"recipe field": "field value"})
}

func TestOpenConnectionToMongoDB_NoSSL(t *testing.T) {
	var (
		mongoVersion = "4.4.8"
		db           = "test-db"
		user         = "test-user"
		password     = "test-password"
	)

	mongoServer, err := mim.Start(mongoVersion)
	if err != nil {
		t.Fatalf("failed to start mongo server: %v", err)
	}
	defer mongoServer.Stop()

	setupMongoConnectionTest(t, mongoServer, db, user, password)
	connectionConfig := &mongoDriver.MongoConnectionConfig{
		ConnectTimeoutInSeconds: 5,
		QueryTimeoutInSeconds:   5,

		Username:        user,
		Password:        password,
		ClusterEndpoint: fmt.Sprintf("localhost:%d", mongoServer.Port()),
		Database:        db,
		Collection:      "testCollection",
	}

	Convey("When a connection to mongodb is attempted", t, func() {

		_, err := mongoDriver.Open(connectionConfig)

		Convey("Then a valid connection (and ping) should be made without error", func() {
			So(err, ShouldBeNil)
		})
	})
}

func TestMongoTLSConnectionConfig(t *testing.T) {
	Convey("When TLS if off", t, func() {
		TLSConfig := &mongoDriver.TLSConnectionConfig{}
		cfg, err := TLSConfig.GetTLSConfig()

		So(err, ShouldBeNil)
		So(cfg, ShouldBeNil)

		Convey("No matter what other attributes are set", func() {
			TLSConfig = &mongoDriver.TLSConnectionConfig{VerifyCert: true, CACertChain: "shouldn't be read"}
			cfg, err = TLSConfig.GetTLSConfig()

			So(err, ShouldBeNil)
			So(cfg, ShouldBeNil)
		})
	})

	Convey("When TLS if on, but we don't verify server certificates", t, func() {
		TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: false}
		cfg, err := TLSConfig.GetTLSConfig()

		So(err, ShouldBeNil)
		So(cfg, ShouldResemble, &tls.Config{InsecureSkipVerify: true})

		Convey("No matter what other attributes are set", func() {
			TLSConfig = &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: false, CACertChain: "shouldn't be read"}
			cfg, err = TLSConfig.GetTLSConfig()

			So(err, ShouldBeNil)
			So(cfg, ShouldResemble, &tls.Config{InsecureSkipVerify: true})
		})
	})

	Convey("When TLS if on and we verify server certificates", t, func() {

		Convey("but we don't supply any CA certs to do the verification", func() {
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeError, mongoDriver.NoCACertChain)
			So(cfg, ShouldBeNil)
		})

		Convey("but we can't read the CA certs to do the verification", func() {
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true, CACertChain: "invalid-file"}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeError, mongoDriver.NoCACertChain)
			So(cfg, ShouldBeNil)
		})

		Convey("but the CA certs are invalid", func() {
			f, e := filepath.Abs("./test/data/invalid.pem")
			if e != nil {
				t.Errorf("error accessing ./test/data/invalid.pem as an invalid cert file: %v", e)
			}
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true, CACertChain: f}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeError, mongoDriver.InvalidCACertChain)
			So(cfg, ShouldBeNil)
		})

		Convey("and the CA certs are valid", func() {
			f, e := filepath.Abs("./test/data/rds-combined-ca-bundle.pem")
			if e != nil {
				t.Errorf("error accessing ./test/data/rds-combined-ca-bundle.pem as a valid cert file: %v", e)
			}
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true, CACertChain: f}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeNil)
			So(cfg, ShouldNotBeNil)
		})
	})
}

func TestMongoConnectionConfig_GetConnectionURIWhen(t *testing.T) {
	connectionConfig := &mongoDriver.MongoConnectionConfig{
		ClusterEndpoint: "localhost:27017",
		Database:        "test-db",
	}

	Convey("When Credentials Are Present", t, func() {
		connectionConfig.Username = "test-user"
		connectionConfig.Password = "test-pass"
		So(connectionConfig.GetConnectionURI(), ShouldEqual, "mongodb://test-user:test-pass@localhost:27017/test-db")
	})

	Convey("When Credentials Are Not Configured", t, func() {
		connectionConfig.Username = ""
		connectionConfig.Password = ""
		So(connectionConfig.GetConnectionURI(), ShouldEqual, "mongodb://localhost:27017/test-db")
	})
}

func setupMongoConnectionTest(t *testing.T, mongoServer *mim.Server, db, user, password string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoServer.URI()))
	if err != nil {
		t.Fatalf("failed to connect to mongo server: %v", err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = client.Database(db).RunCommand(ctx, bson.D{{"createUser", user}, {"pwd", password}, {"roles", []bson.M{}}}).Err()
	if err != nil {
		t.Fatalf("couldn't set up test: %v", err)
	}

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
