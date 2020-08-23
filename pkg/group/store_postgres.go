package group

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/bytearray"
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

func (s *PostgreSQLStore) oneGroup(ctx context.Context, q string, args ...interface{}) (g Group, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&g.ID, &g.ParentID, &g.DisplayName, &g.Key, &g.Flags)

	switch err {
	case nil:
		return g, nil
	case pgx.ErrNoRows:
		return g, ErrGroupNotFound
	default:
		return g, errors.Wrap(err, "failed to scan group")
	}
}

func (s *PostgreSQLStore) manyGroups(ctx context.Context, q string, args ...interface{}) (gs []Group, err error) {
	gs = make([]Group, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch groups")
	}
	defer rows.Close()

	for rows.Next() {
		var g Group

		if err = rows.Scan(&g.ID, &g.ParentID, &g.DisplayName, &g.Key, &g.Flags); err != nil {
			return gs, errors.Wrap(err, "failed to scan groups")
		}

		gs = append(gs, g)
	}

	return gs, nil
}

func (s *PostgreSQLStore) oneRelation(ctx context.Context, q string, args ...interface{}) (rel Relation, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&rel.GroupID, &rel.Asset.Kind, &rel.Asset.ID)

	switch err {
	case nil:
		return rel, nil
	case pgx.ErrNoRows:
		return rel, ErrRelationNotFound
	default:
		return rel, errors.Wrap(err, "failed to scan relation")
	}
}

func (s *PostgreSQLStore) manyRelations(ctx context.Context, q string, args ...interface{}) (relations []Relation, err error) {
	relations = make([]Relation, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return relations, errors.Wrap(err, "failed to fetch relations")
	}
	defer rows.Close()

	for rows.Next() {
		var rel Relation

		if err = rows.Scan(&rel.GroupID, &rel.Asset.Kind, &rel.Asset.ID); err != nil {
			return relations, errors.Wrap(err, "failed to scan relations")
		}

		relations = append(relations, rel)
	}

	return relations, nil
}

func (s *PostgreSQLStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	if g.ID == uuid.Nil {
		return g, ErrNilGroupID
	}

	q := `
	INSERT INTO "group"(id, parent_id, name, key, flags) 
	VALUES($1, $2, $3, $4, $5)
	ON CONFLICT ON CONSTRAINT group_pk
	DO UPDATE 
		SET parent_id 	= EXCLUDED.parent_id,
			name		= EXCLUDED.name,
			key			= EXCLUDED.key,
			flags		= EXCLUDED.flags`

	_, err := s.db.ExecEx(
		ctx,
		q,
		nil,
		g.ID, g.ParentID, g.DisplayName, g.Key, g.Flags,
	)

	if err != nil {
		return g, errors.Wrap(err, "failed to execute insert statement")
	}

	return g, nil
}

func (s *PostgreSQLStore) CreateRelation(ctx context.Context, rel Relation) (err error) {
	if rel.GroupID == uuid.Nil {
		return ErrNilGroupID
	}

	if rel.Asset.ID == uuid.Nil {
		return ErrNilAssetID
	}

	q := `
	INSERT INTO group_assets(group_id, asset_kind, asset_id) 
	VALUES($1, $2, $3)
	ON CONFLICT ON CONSTRAINT group_assets_pk 
	DO NOTHING
	`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		rel.GroupID, rel.Asset.Kind, rel.Asset.ID,
	)

	if err != nil {
		return errors.Wrap(err, "failed to execute insert statement")
	}

	return nil
}

func (s *PostgreSQLStore) FetchGroupByID(ctx context.Context, groupID uuid.UUID) (Group, error) {
	return s.oneGroup(ctx, `SELECT id, parent_id, name, key, flags FROM "group" WHERE id = $1 LIMIT 1`, groupID)
}

func (s *PostgreSQLStore) FetchGroupByKey(ctx context.Context, key string) (Group, error) {
	return s.oneGroup(ctx, `SELECT id, parent_id, name, key, flags FROM "group" WHERE key = $1 LIMIT 1`, key)
}

func (s *PostgreSQLStore) FetchGroupByName(ctx context.Context, name string) (g Group, err error) {
	return s.oneGroup(ctx, `SELECT id, parent_id, name, key, flags FROM "group" WHERE name $1 LIMIT 1`, name)
}

func (s *PostgreSQLStore) FetchGroupsByName(ctx context.Context, isPartial bool, name string) (gs []Group, err error) {
	if isPartial {
		return s.manyGroups(ctx, `SELECT id, parent_id, name, key, flags FROM "group" WHERE name = '%' || $1 || '%'`, name)
	}

	return s.manyGroups(ctx, `SELECT id, parent_id, name, key, flags FROM "group" WHERE name = $1`, name)
}

func (s *PostgreSQLStore) FetchAllGroups(ctx context.Context) (gs []Group, err error) {
	return s.manyGroups(ctx, `SELECT id, parent_id, name, key, flags FROM "group"`)
}

func (s *PostgreSQLStore) FetchAllRelations(ctx context.Context) (relations []Relation, err error) {
	return s.manyRelations(ctx, `SELECT group_id, asset_kind, asset_id FROM group_assets`)
}

func (s *PostgreSQLStore) FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]Relation, error) {
	return s.manyRelations(ctx, `SELECT group_id, asset_kind, asset_id FROM group_assets WHERE group_id = $1`, groupID)
}

func (s *PostgreSQLStore) HasRelation(ctx context.Context, rel Relation) (bool, error) {
	q := `
	SELECT group_id, asset_kind, asset_id 
	FROM group_assets
	WHERE 
		group_id		= $1 
		AND asset_kind	= $2 
		AND asset_id	= $3
	LIMIT 1`

	_rel, err := s.oneRelation(ctx, q, rel.GroupID, rel.Asset.Kind, rel.Asset.ID)
	if err != nil {
		if errors.Cause(err) == ErrRelationNotFound {
			return false, nil
		}

		return false, err
	}

	return rel == _rel, nil
}

func (s *PostgreSQLStore) DeleteByID(ctx context.Context, groupID uuid.UUID) (err error) {
	_, err = s.db.ExecEx(ctx, `DELETE FROM "group" WHERE id = $1`, nil, groupID)
	if err != nil {
		return errors.Wrap(err, "failed to delete group")
	}

	return nil
}

func (s *PostgreSQLStore) DeleteRelation(ctx context.Context, rel Relation) (err error) {
	q := `
	DELETE FROM group_assets 
	WHERE 
		group_id		= $1 
		AND asset_kind	= $2 
		AND asset_id	= $3`

	_, err = s.db.ExecEx(ctx, q, nil, rel.GroupID, rel.Asset.Kind, rel.Asset.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete group relation")
	}

	return nil
}
