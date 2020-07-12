package database

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/gocraft/dbr/v2"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/pkg/errors"
)

var mysqlConn *dbr.Connection

// MySQLConnection returns database singleton instance
func MySQLConnection() *dbr.Connection {
	// using a package global variable
	if mysqlConn == nil {
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

		mysqlConn = conn
	}

	return mysqlConn
}

// MySQLForTesting simply returns a database mysqlConn
func MySQLForTesting() (conn *dbr.Connection, err error) {
	if !util.IsTestMode() {
		log.Fatal("TruncateTestDatabase() can only be called during testing")
		return nil, nil
	}

	conn, err = dbr.Open("mysql", os.Getenv("HOMETOWN_TEST_DATABASE"), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to test database")
	}

	tx, err := conn.NewSession(nil).Begin()
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUnlessCommitted()

	// temporarily disabling foreign key checks to enable truncate
	if _, err = tx.Exec("SET foreign_key_checks = 0"); err != nil {
		panic(err)
	}

	defer func() {
		if p := recover(); p != nil {
			err = errors.Wrap(err, "recovering from panic after TruncateDatabaseForTesting")
		}
	}()

	tables := []string{
		"user",
		"user_email",
		"user_phone",
		"user_profile",
		"password",
		"token",
		"group",
		"group_users",
		"accesspolicy",
		"accesspolicy_roster",
	}

	// truncating tables
	for _, tableName := range tables {
		_, err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE `%s`", tableName))
		if err != nil {
			return nil, errors.Wrap(err, tx.Rollback().Error())
		}
	}

	if _, err = tx.Exec("SET foreign_key_checks = 1"); err != nil {
		panic(err)
	}

	if err := tx.Commit(); err != nil {
		panic(err)
	}

	return conn, nil
}
