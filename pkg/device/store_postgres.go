package client

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type SQLStore struct {
	db *pgx.Conn
}

func NewSQLStore(db *pgx.Conn) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &SQLStore{db}, nil
}

func (s *SQLStore) oneDevice(ctx context.Context, q string, args ...interface{}) (d Device, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&d.ID, &d.Name, &d.IMEI, &d.MEID, &d.SerialNumber, &d.Flags, &d.RegisteredAt, &d.ExpireAt)

	switch err {
	case nil:
		return d, nil
	case pgx.ErrNoRows:
		return d, ErrDeviceNotFound
	default:
		return d, errors.Wrap(err, "failed to scan device")
	}
}

func (s *SQLStore) manyDevices(ctx context.Context, q string, args ...interface{}) (ds []Device, err error) {
	ds = make([]Device, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch devices")
	}
	defer rows.Close()

	for rows.Next() {
		var d Device

		if err = rows.Scan(&d.ID, &d.Name, &d.IMEI, &d.MEID, &d.SerialNumber, &d.Flags, &d.RegisteredAt, &d.ExpireAt); err != nil {
			return ds, errors.Wrap(err, "failed to scan devices")
		}

		ds = append(ds, d)
	}

	return ds, nil
}

func (s *SQLStore) oneRelation(ctx context.Context, q string, args ...interface{}) (rel Relation, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&rel.DeviceID, &rel.Asset.Kind, &rel.Asset.ID)

	switch err {
	case nil:
		return rel, nil
	case pgx.ErrNoRows:
		return rel, ErrRelationNotFound
	default:
		return rel, errors.Wrap(err, "failed to scan relation")
	}
}

func (s *SQLStore) manyRelations(ctx context.Context, q string, args ...interface{}) (relations []Relation, err error) {
	relations = make([]Relation, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return relations, errors.Wrap(err, "failed to fetch relations")
	}
	defer rows.Close()

	for rows.Next() {
		var rel Relation

		if err = rows.Scan(&rel.DeviceID, &rel.Asset, &rel.Asset.ID); err != nil {
			return relations, errors.Wrap(err, "failed to scan relations")
		}

		relations = append(relations, rel)
	}

	return relations, nil
}

func (s *SQLStore) UpsertDevice(ctx context.Context, d Device) (_ Device, err error) {
	if d.ID == uuid.Nil {
		return d, ErrInvalidDeviceID
	}

	q := `
	INSERT INTO device(id, name, imei, meid, serial_number, flags, registered_at, expire_at) 
	VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT ON CONSTRAINT device_pk
	DO UPDATE 
		SET name			= EXCLUDED.name,
			imei			= EXCLUDED.imei,
			meid			= EXCLUDED.meid,
			serial_number	= EXCLUDED.serial_number,
			flags			= EXCLUDED.flags,
			registered_at	= EXCLUDED.registered_at,
			expire_at		= EXCLUDED.expire_at`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		d.ID, d.Name, d.IMEI, d.MEID, d.SerialNumber, d.Flags, d.RegisteredAt, d.ExpireAt,
	)

	if err != nil {
		return d, errors.Wrap(err, "failed to execute insert statement")
	}

	return d, nil
}

func (s *SQLStore) CreateRelation(ctx context.Context, rel Relation) (err error) {
	if rel.DeviceID == uuid.Nil {
		return ErrInvalidDeviceID
	}

	if rel.Asset.ID == uuid.Nil {
		return ErrInvalidAssetID
	}

	q := `
	INSERT INTO device_assets(device_id, asset_kind, asset_id) 
	VALUES($1, $2, $3)
	ON CONFLICT ON CONSTRAINT device_assets_pk 
	DO NOTHING`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		rel.DeviceID, rel.Asset.Kind, rel.Asset.ID,
	)

	if err != nil {
		return errors.Wrap(err, "failed to execute insert statement")
	}

	return nil
}

func (s *SQLStore) FetchDeviceByID(ctx context.Context, groupID uuid.UUID) (Device, error) {
	return s.oneDevice(ctx, `SELECT id, name, kind, flags, registered_at, expire_at FROM client WHERE id = $1 LIMIT 1`, groupID)
}

func (s *SQLStore) FetchAllDevices(ctx context.Context) ([]Device, error) {
	return s.manyDevices(ctx, `SELECT id, name, kind, flags, registered_at, expire_at FROM client`)
}

func (s *SQLStore) FetchAllRelations(ctx context.Context) (relations []Relation, err error) {
	return s.manyRelations(ctx, `SELECT device_id, asset_kind, asset_id FROM device_assets`)
}

func (s *SQLStore) FetchClientRelations(ctx context.Context, groupID uuid.UUID) ([]Relation, error) {
	return s.manyRelations(ctx, `SELECT device_id, asset_kind, asset_id FROM device_assets WHERE group_id = $1`, groupID)
}

func (s *SQLStore) HasRelation(ctx context.Context, rel Relation) bool {
	q := `
	SELECT device_id, asset_kind, asset_id
	FROM device_assets 
	WHERE 
		device_id = $1 AND asset_kind = $2 AND asset_id = $3 
	LIMIT 1`

	_rel, err := s.oneRelation(ctx, q, rel.DeviceID, rel.Asset.Kind, rel.Asset.ID)
	if err != nil {
		if errors.Cause(err) == ErrRelationNotFound {
			return false
		}

		return false
	}

	return rel == _rel
}

func (s *SQLStore) DeleteDeviceByID(ctx context.Context, deviceID uuid.UUID) (err error) {
	_, err = s.db.ExecEx(ctx, `DELETE FROM device WHERE id = $1`, nil, deviceID)
	if err != nil {
		return errors.Wrap(err, "failed to delete device")
	}

	return nil
}

func (s *SQLStore) DeleteRelation(ctx context.Context, rel Relation) (err error) {
	q := `
	DELETE FROM device_assets 
	WHERE 
		group_id = $1 AND asset_kind = $2 AND asset_id = $3`

	_, err = s.db.ExecEx(ctx, q, nil, rel.DeviceID, rel.Asset.Kind, rel.Asset.ID)
	if err != nil {
		return errors.Wrap(err, "failed to delete device relation")
	}

	return nil
}
