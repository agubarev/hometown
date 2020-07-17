package group

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type PostgreSQLStore struct {
	db *pgx.Conn
}

func NewPostgreSQLStore(db *pgx.Conn) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &PostgreSQLStore{db}, nil
}

func (s *PostgreSQLStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) CreateRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) error {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchGroupByID(ctx context.Context, groupID uuid.UUID) (g Group, err error) {
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
	rows, err := s.db.QueryEx(ctx, `SELECT id, parent_id, name, key, flags FROM "group"`, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch all groups")
	}

	for rows.Next() {
		var g Group

		if err = rows.Scan(&g.ID, &g.ParentID, &g.DisplayName, &g.Key, &g.Flags); err != nil {
			return nil, errors.Wrap(err, "failed to scan values")
		}

		gs = append(gs, g)
	}

	return gs, nil
}

func (s *PostgreSQLStore) FetchAllRelations(ctx context.Context) (relations []Relation, err error) {
	rows, err := s.db.QueryEx(ctx, `SELECT group_id, asset_id, asset_kind FROM "group_assets"`, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch all groups")
	}

	for rows.Next() {
		var rel Relation

		if err = rows.Scan(&rel.GroupID, &rel.Asset.ID, &rel.Asset.Kind); err != nil {
			return nil, errors.Wrap(err, "failed to scan values")
		}

		relations = append(relations, rel)
	}

	return relations, nil
}

func (s *PostgreSQLStore) FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) HasRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) (bool, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) DeleteByID(ctx context.Context, groupID uuid.UUID) error {
	panic("implement me")
}

func (s *PostgreSQLStore) DeleteRelation(ctx context.Context, groupID uuid.UUID, assetKind AssetKind, assetID uuid.UUID) error {
	panic("implement me")
}
