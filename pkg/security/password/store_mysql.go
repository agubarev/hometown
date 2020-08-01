package password

/*
type MySQLStore struct {
	db *pgx.Conn
}

// NewPostgreSQLStore initializes a default password store
func NewPostgreSQLStore(db *pgx.Conn) (Store, error) {
	// reserving error return for the future, just in case
	return &MySQLStore{db}, nil
}

// Upsert stores password
// ObjectID must be equal to the user's ObjectID
func (s *MySQLStore) Upsert(ctx context.Context, p Password) (err error) {
	if p.OwnerID == 0 {
		return ErrNilOwnerID
	}

	if p.Hash[0] == 0 {
		return ErrEmptyPassword
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	_, err = s.db.NewSession(nil).
		ExecContext(
			ctx,
			"INSERT INTO `password`(`kind`, `owner_id`, `hash`, `is_change_required`, `created_at`, `expire_at`) VALUES(?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE `hash`=?, `updated_at`=?, `expire_at`=?",
			p.OwnerKind, p.OwnerID, p.Hash, p.IsChangeRequired, p.CreatedAt, p.ExpireAt, p.Hash, p.UpdatedAt, p.ExpireAt,
		)

	if err != nil {
		return errors.Wrap(err, "failed to upsert password")
	}

	return nil
}

// UpdatePolicy updates an existing password record
func (s *MySQLStore) Update(ctx context.Context, k OwnerKind, ownerID uint32, newpass []byte) (err error) {
	if len(newpass) == 0 {
		return ErrEmptyPassword
	}

	updates := map[string]interface{}{
		"hash": newpass,
	}

	// updating the password
	_, err = s.db.NewSession(nil).Update("password").
		SetMap(updates).Where("kind = ? AND owner_id = ?", k, ownerID).ExecContext(ctx)

	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a stored password
func (s *MySQLStore) Get(ctx context.Context, k OwnerKind, userID uint32) (p Password, err error) {
	// retrieving password
	err = s.db.NewSession(nil).
		Select("*").
		From("password").
		Where("kind = ? AND owner_id = ?", k, userID).
		LoadOneContext(ctx, &p)

	if err != nil {
		if err == dbr.ErrNotFound {
			return p, ErrPasswordNotFound
		}

		return p, err
	}

	return p, nil
}

// DeletePolicy deletes a stored password
func (s *MySQLStore) Delete(ctx context.Context, k OwnerKind, ownerID uint32) (err error) {
	_, err = s.db.NewSession(nil).
		DeleteFrom("password").
		Where("kind = ? AND owner_id = ?", k, ownerID).
		ExecContext(ctx)

	return err
}
*/
