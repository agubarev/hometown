package group

import "github.com/gocql/gocql"

type CassandraStore struct {
	conn gocql.Conn
}
