package password

import (
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// errors
var (
	ErrNilOwnerID       = errors.New("owner id is zero")
	ErrZeroKind         = errors.New("password kind is zero")
	ErrNilPasswordStore = errors.New("password store is nil")
	ErrEmptyPassword    = errors.New("empty password is forbidden")
	ErrPasswordNotFound = errors.New("password not found")
	ErrShortPassword    = errors.New("password is too short")
	ErrLongPassword     = errors.New("password is too long")
	ErrUnsafePassword   = errors.New("password is too unsafe")
)

// userManager describes the behaviour of a user password manager
type Manager interface {
	Upsert(ctx context.Context, p Password) error
	Get(ctx context.Context, o Owner) (p Password, err error)
	Delete(ctx context.Context, o Owner) error
}

type defaultManager struct {
	store Store
}

// NewManager initializes the default user password manager
func NewManager(store Store) (Manager, error) {
	if store == nil {
		return nil, ErrNilPasswordStore
	}

	pm := &defaultManager{
		store: store,
	}

	return pm, nil
}

func (m *defaultManager) Upsert(ctx context.Context, p Password) (err error) {
	if m.store == nil {
		return ErrNilPasswordStore
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	return m.store.Upsert(ctx, p)
}

func (m *defaultManager) Get(ctx context.Context, o Owner) (p Password, err error) {
	if m.store == nil {
		return p, ErrNilPasswordStore
	}

	if o.ID == uuid.Nil {
		return p, ErrNilOwnerID
	}

	return m.store.Get(ctx, o)
}

func (m *defaultManager) Delete(ctx context.Context, o Owner) error {
	if m.store == nil {
		return ErrNilPasswordStore
	}

	if o.ID == uuid.Nil {
		return ErrNilOwnerID
	}

	return m.store.Delete(ctx, o)
}
