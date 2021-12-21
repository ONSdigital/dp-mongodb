package mongodb_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
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
		Username:        "XXX- username for recipe-api for authentication",
		Password:        "XXX - the password for the above username",

		ConnectTimeout:                5 * time.Second,
		QueryTimeout:                  5 * time.Second,
		IsStrongReadConcernEnabled:    true,
		IsWriteConcernMajorityEnabled: true,
		TLSConnectionConfig: mongoDriver.TLSConnectionConfig{
			IsSSL:              true,
			VerifyCert:         true,
			CACertChain:        "-----BEGIN CERTIFICATE-----ACTUAL CERT CONTENTS -----END CERTIFICATE-----",
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
	_, _ = mongoDB.Collection("recipes").Insert(ctx, bson.M{"recipe field": "field value"})
}

func TestOpenConnectionToMongoDB_NoSSL(t *testing.T) {
	Convey("Given a mongodb server is running", t, func() {

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
			ConnectTimeout: 5 * time.Second,
			QueryTimeout:   5 * time.Second,

			Username:        user,
			Password:        password,
			ClusterEndpoint: fmt.Sprintf("localhost:%d", mongoServer.Port()),
			Database:        db,
		}

		Convey("When a connection is attempted", func() {
			conn, err := mongoDriver.Open(connectionConfig)

			Convey("Then a valid connection should be made without error", func() {
				So(err, ShouldBeNil)
				So(conn, ShouldNotBeNil)
				Convey("And it pings", func() {
					pingErr := conn.Ping(context.Background(), 2*time.Second)
					So(pingErr, ShouldBeNil)
				})
			})
		})

		Convey("When a connection is attempted using the mongodb:// schema for the endpoint", func() {
			connectionConfig.ClusterEndpoint = fmt.Sprintf("mongodb://localhost:%d", mongoServer.Port())
			conn, err := mongoDriver.Open(connectionConfig)

			Convey("Then a valid connection should be made without error", func() {
				So(err, ShouldBeNil)
				So(conn, ShouldNotBeNil)

				Convey("And it pings", func() {
					pingErr := conn.Ping(context.Background(), 2*time.Second)
					So(pingErr, ShouldBeNil)
				})
			})
		})

		Convey("When a connection is attempted using an invalid endpoint", func() {
			connectionConfig.ClusterEndpoint = fmt.Sprintf("mysql://localhost:%d", mongoServer.Port())
			_, err := mongoDriver.Open(connectionConfig)

			Convey("Then an error is returned", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, fmt.Sprintf("invalid mongodb address: %s", connectionConfig.ClusterEndpoint))
			})
		})
	})
}

func TestMongoTLSConnectionConfig(t *testing.T) {
	Convey("When TLS is off", t, func() {
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

	Convey("When TLS is on, but we don't verify server certificates", t, func() {
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

	Convey("When TLS is on and we verify server certificates", t, func() {

		Convey("but we don't supply any CA certs to do the verification", func() {
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeError, mongoDriver.NoCACertChain)
			So(cfg, ShouldBeNil)
		})

		Convey("but the CA certs are invalid", func() {
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true, CACertChain: "invalid certificate"}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeError, mongoDriver.InvalidCACertChain)
			So(cfg, ShouldBeNil)
		})

		Convey("and the CA certs are valid", func() {
			c, e := ioutil.ReadFile("./test/data/rds-combined-ca-bundle.pem")
			if e != nil {
				t.Errorf("error accessing ./test/data/rds-combined-ca-bundle.pem as a valid cert file: %v", e)
			}
			TLSConfig := &mongoDriver.TLSConnectionConfig{IsSSL: true, VerifyCert: true, CACertChain: string(c)}
			cfg, err := TLSConfig.GetTLSConfig()
			So(err, ShouldBeNil)
			So(cfg, ShouldNotBeNil)
		})
	})
}

func TestMongoConnectionConfig_GetConnectionURIWhen(t *testing.T) {
	Convey("Given a MongoConnectionConfig", t, func() {
		connectionConfig := &mongoDriver.MongoConnectionConfig{
			Database: "test-db",
		}

		Convey("When the endpoint does not include a scheme", func() {
			connectionConfig.ClusterEndpoint = "localhost:27017"

			Convey("And credentials are present", func() {
				connectionConfig.Username = "test-user"
				connectionConfig.Password = "test-pass"

				Convey("The connection URI is created correctly", func() {
					uri, err := connectionConfig.GetConnectionURI()
					So(err, ShouldBeNil)
					So(uri, ShouldEqual, "mongodb://test-user:test-pass@localhost:27017/test-db?directConnection=true")
				})
			})

			Convey("And credentials are not configured", func() {
				connectionConfig.Username = ""
				connectionConfig.Password = ""

				Convey("The connection URI is created correctly", func() {
					uri, err := connectionConfig.GetConnectionURI()
					So(err, ShouldBeNil)
					So(uri, ShouldEqual, "mongodb://localhost:27017/test-db?directConnection=true")
				})
			})
		})

		Convey("When the endpoint does include the mongodb scheme", func() {
			connectionConfig.ClusterEndpoint = "mongodb://localhost:27017"

			Convey("And credentials are present", func() {
				connectionConfig.Username = "test-user"
				connectionConfig.Password = "test-pass"

				Convey("And a replica set is configured ", func() {
					connectionConfig.ReplicaSet = "repl0"

					Convey("The connection URI is created correctly", func() {
						uri, err := connectionConfig.GetConnectionURI()
						So(err, ShouldBeNil)
						So(uri, ShouldEqual, "mongodb://test-user:test-pass@localhost:27017/test-db?replicaSet=repl0")
					})
				})
			})

			Convey("And credentials are not configured", func() {
				connectionConfig.Username = ""
				connectionConfig.Password = ""

				Convey("And a replica set is not set", func() {

					Convey("The connection URI is created correctly", func() {
						uri, err := connectionConfig.GetConnectionURI()
						So(err, ShouldBeNil)
						So(uri, ShouldEqual, "mongodb://localhost:27017/test-db?directConnection=true")
					})
				})
			})
		})

		Convey("When the endpoint uses an invalid scheme", func() {
			connectionConfig.ClusterEndpoint = "mysql://localhost:27017"
			Convey("The connection URI returns an error", func() {
				_, err := connectionConfig.GetConnectionURI()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldEqual, "invalid mongodb address: mysql://localhost:27017")
			})

		})
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
	err = client.Database(db).RunCommand(ctx, bson.D{{Key: "createUser", Value: user}, {Key: "pwd", Value: password}, {Key: "roles", Value: []bson.M{}}}).Err()
	if err != nil {
		t.Fatalf("couldn't set up test: %v", err)
	}

}
