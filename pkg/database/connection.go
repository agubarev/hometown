package database

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/gocraft/dbr/v2"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var connection *dbr.Connection

// Instance returns database singleton instance
func Instance() *dbr.Connection {
	// using a package global variable
	if connection == nil {
		// checking whether it's called during `go test`
		testMode := flag.Lookup("test.v") != nil

		dsn := os.Getenv("HOMETOWN_DATABASE")
		if testMode {
			dsn = os.Getenv("HOMETOWN_TEST_DATABASE")
		}

		conn, err := dbr.Open("mysql", strings.TrimSpace(dsn), nil)
		if err != nil {
			log.Fatalf("failed to connect to database: %s", err)
		}

		connection = conn
	}

	return connection
}

func TruncateAllDatabaseData(conn *dbr.Connection) {
	/*
		if flag.Lookup("test.v") == nil {
			log.Fatal("TruncateTestDatabase() can only be called during testing")
			return
		}
	*/

	sess := conn.NewSession(nil)

	// temporarily disabling foreign key checks to enable truncate
	sess.Exec("SET foreign_key_checks = 0")
	defer func() {
		sess.Exec("SET foreign_key_checks = 1")
	}()

	//sess.Exec("truncate table ...")
}
