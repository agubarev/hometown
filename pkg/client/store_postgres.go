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

func (s *SQLStore) oneClient(ctx context.Context, q string, args ...interface{}) (c *Client, err error) {
	c = &Client{}

	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&c.ID, &c.Name, &c.Flags, &c.RegisteredAt, &c.ExpireAt, &c.URLs, &c.entropy)

	switch err {
	case nil:
		return c, nil
	case pgx.ErrNoRows:
		return c, ErrClientNotFound
	default:
		return c, errors.Wrap(err, "failed to scan client")
	}
}

func (s *SQLStore) manyClients(ctx context.Context, q string, args ...interface{}) (cs []*Client, err error) {
	cs = make([]*Client, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch clients")
	}
	defer rows.Close()

	for rows.Next() {
		c := &Client{}

		if err = rows.Scan(&c.ID, &c.Name, &c.Flags, &c.RegisteredAt, &c.ExpireAt, &c.URLs, &c.entropy); err != nil {
			return cs, errors.Wrap(err, "failed to scan client")
		}

		cs = append(cs, c)
	}

	return cs, nil
}

func (s *SQLStore) UpsertClient(ctx context.Context, c *Client) (_ *Client, err error) {
	if c.ID == uuid.Nil {
		return c, ErrInvalidClientID
	}

	q := `
	INSERT INTO client(id, name, flags, registered_at, expire_at, urls, entropy) 
	VALUES($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT ON CONSTRAINT client_pk
	DO UPDATE 
		SET name			= EXCLUDED.name,
			kind			= EXCLUDED.kind,
			flags			= EXCLUDED.flags,
			expire_at		= EXCLUDED.expire_at,
			urls			= EXCLUDED.urls,
			entropy			= EXCLUDED.entropy`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		c.ID, c.Name, c.Flags, c.RegisteredAt, c.ExpireAt, c.URLs, c.entropy,
	)

	if err != nil {
		return c, errors.Wrap(err, "failed to execute insert statement")
	}

	return c, nil
}

func (s *SQLStore) FetchClientByID(ctx context.Context, clientID uuid.UUID) (*Client, error) {
	return s.oneClient(ctx, `SELECT id, name, flags, registered_at, expire_at, urls, entropy FROM client WHERE id = $1 LIMIT 1`, clientID)
}

func (s *SQLStore) FetchAllClients(ctx context.Context) (gs []*Client, err error) {
	return s.manyClients(ctx, `SELECT id, name, flags, registered_at, expire_at, urls, entropy FROM client`)
}

func (s *SQLStore) DeleteClientByID(ctx context.Context, clientID uuid.UUID) (err error) {
	_, err = s.db.ExecEx(ctx, `DELETE FROM client WHERE id = $1`, nil, clientID)
	if err != nil {
		return errors.Wrap(err, "failed to delete client")
	}

	return nil
}
