package group

import (
	"context"

	"github.com/jackc/pgx"
)

type PostgreSQLStore struct {
	db *pgx.Conn
}

func (s *PostgreSQLStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) CreateRelation(ctx context.Context, groupID, userID uint32) error {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupByID(ctx context.Context, groupID uint32) (g Group, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupByKey(ctx context.Context, key TKey) (g Group, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupByName(ctx context.Context, name TName) (g Group, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupsByName(ctx context.Context, isPartial bool, name TName) (gs []Group, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchAllGroups(ctx context.Context) (gs []Group, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchAllRelations(ctx context.Context) (map[uint32][]uint32, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupRelations(ctx context.Context, groupID uint32) ([]uint32, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) HasRelation(ctx context.Context, groupID, userID uint32) (bool, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) DeleteByID(ctx context.Context, groupID uint32) error {
	panic("implement me")
}

func (s *PostgreSQLStore) DeleteRelation(ctx context.Context, groupID, userID uint32) error {
	panic("implement me")
}
