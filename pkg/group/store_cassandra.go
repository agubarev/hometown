package group

import (
	"context"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

type CassandraStore struct {
	conn *gocql.Conn
}

func (c CassandraStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	panic("implement me")
}

func (c CassandraStore) CreateRelation(ctx context.Context, rel Relation) error {
	panic("implement me")
}

func (c CassandraStore) FetchGroupByID(ctx context.Context, groupID uuid.UUID) (g Group, err error) {
	panic("implement me")
}

func (c CassandraStore) FetchGroupByKey(ctx context.Context, key string) (g Group, err error) {
	panic("implement me")
}

func (c CassandraStore) FetchGroupByName(ctx context.Context, name string) (g Group, err error) {
	panic("implement me")
}

func (c CassandraStore) FetchGroupsByName(ctx context.Context, isPartial bool, name string) (gs []Group, err error) {
	panic("implement me")
}

func (c CassandraStore) HasRelation(ctx context.Context, rel Relation) (bool, error) {
	panic("implement me")
}

func (c CassandraStore) FetchAllGroups(ctx context.Context) (gs []Group, err error) {
	panic("implement me")
}

func (c CassandraStore) FetchAllRelations(ctx context.Context) ([]Relation, error) {
	panic("implement me")
}

func (c CassandraStore) FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]Relation, error) {
	panic("implement me")
}

func (c CassandraStore) DeleteByID(ctx context.Context, groupID uuid.UUID) error {
	panic("implement me")
}

func (c CassandraStore) DeleteRelation(ctx context.Context, rel Relation) error {
	panic("implement me")
}
