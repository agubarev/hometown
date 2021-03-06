package token

/*
type tokenStore struct {
	db *dbr.Connection
}

// NewStore initializes and returns a new default token store
func NewStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &tokenStore{db}, nil
}

// UpdatePolicy puts token into a store
func (s *tokenStore) Put(ctx context.Context, t *Hash) error {
	if t == nil {
		return ErrEmptyTokenHash
	}

	_, err := s.db.NewSession(nil).
		InsertInto("token").
		Columns(guard.DBColumnsFrom(t)...).
		Record(t).
		ExecContext(ctx)

	if err != nil {
		return err
	}

	return nil
}

// Get retrieves token from a store
func (s *tokenStore) Get(ctx context.Context, token string) (*Hash, error) {
	t := new(Hash)

	err := s.db.NewSession(nil).
		SelectBySql("SELECT * FROM token WHERE token = ? LIMIT 1", token).
		LoadOneContext(ctx, t)

	if err != nil {
		if err == dbr.ErrNotFound {
			return nil, ErrTokenNotFound
		}

		return nil, err
	}

	return t, nil
}

// DeletePolicy deletes token from a store
func (s *tokenStore) Delete(ctx context.Context, token string) error {
	result, err := s.db.NewSession(nil).
		DeleteFrom("token").
		Where("token = ?", token).
		ExecContext(ctx)

	if err != nil {
		return err
	}

	// checking whether anything was updated at all
	ra, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}
*/
