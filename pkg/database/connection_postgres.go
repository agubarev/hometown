package database

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/log/zapadapter"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var postgresConn *pgx.Conn

// PostgreSQLConnection returns database singleton instance
func PostgreSQLConnection(logger *zap.Logger) *pgx.Conn {
	// using a package global variable
	if postgresConn == nil {
		// checking whether it's called during `go test`
		testMode := flag.Lookup("test.v") != nil

		dsn := os.Getenv("HOMETOWN_DATABASE")

		// better safe than sorry
		if testMode {
			dsn = os.Getenv("HOMETOWN_TEST_DATABASE")
		}

		// mysqlConn config
		conf, err := pgx.ParseDSN(dsn)
		if err != nil {
			log.Fatalf("failed to parse DSN: %s", err)
		}

		// injecting logger into database instance
		if logger != nil {
			conf.Logger = zapadapter.NewLogger(logger)
			conf.LogLevel = pgx.LogLevelDebug
		}

		// initializing connection to postgres database
		conn, err := pgx.Connect(conf)
		if err != nil {
			log.Fatalf("failed to connect to database: %s", err)
		}

		postgresConn = conn
	}

	return postgresConn
}

// PostgreSQLForTesting simply returns a database mysqlConn
func PostgreSQLForTesting(logger *zap.Logger) (conn *pgx.Conn, err error) {
	if !util.IsTestMode() {
		log.Fatal("TruncateTestDatabase() can only be called during testing")
		return nil, nil
	}

	// checking whether it's called during `go test`
	testMode := flag.Lookup("test.v") != nil

	dsn := os.Getenv("HOMETOWN_DATABASE")

	// better safe than sorry
	if testMode {
		dsn = os.Getenv("HOMETOWN_TEST_DATABASE")
	}

	// mysqlConn config
	conf, err := pgx.ParseDSN(dsn)
	if err != nil {
		log.Fatalf("failed to parse DSN: %s", err)
	}

	// injecting logger into database instance
	if logger != nil {
		conf.Logger = zapadapter.NewLogger(logger)
		conf.LogLevel = pgx.LogLevelDebug
	}

	// initializing connection to postgres database
	conn, err = pgx.Connect(conf)
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}

	postgresConn = conn

	tx, err := conn.Begin()
	if err != nil {
		log.Fatalf("failed to begin transaction: %s", err)
	}

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