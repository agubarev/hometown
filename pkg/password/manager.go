package password

import (
	"context"

	"github.com/pkg/errors"
)

// errors
var (
	ErrZeroOwnerID      = errors.New("owner id is zero")
	ErrZeroKind         = errors.New("password kind is zero")
	ErrNilPasswordStore = errors.New("password store is nil")
	ErrNilPassword      = errors.New("password is nil")
	ErrEmptyPassword    = errors.New("empty password is forbidden")
	ErrPasswordNotFound = errors.New("password not found")
	ErrShortPassword    = errors.New("password is too short")
	ErrLongPassword     = errors.New("password is too long")
	ErrUnsafePassword   = errors.New("password is too unsafe")
)

type Kind uint8

// password owner kinds
const (
	KUser Kind = iota
)

// Manager describes the behaviour of a user password manager
type Manager interface {
	Create(ctx context.Context, kind Kind, ownerID int, p *Password) error
	Get(ctx context.Context, kind Kind, ownerID int) (*Password, error)
	Delete(ctx context.Context, kind Kind, ownerID int) error
}

type defaultManager struct {
	store Store
}

// NewDefaultManager initializes the default user password manager
func NewDefaultManager(store Store) (Manager, error) {
	if store == nil {
		return nil, ErrNilPasswordStore
	}

	pm := &defaultManager{
		store: store,
	}

	return pm, nil
}

func (m *defaultManager) Create(ctx context.Context, k Kind, ownerID int, p *Password) (err error) {
	if m.store == nil {
		return ErrNilPasswordStore
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	return m.store.Create(ctx, p)
}

func (m *defaultManager) Get(ctx context.Context, k Kind, ownerID int) (*Password, error) {
	if m.store == nil {
		return nil, ErrNilPasswordStore
	}

	if k == 0 {
		return nil, ErrZeroKind
	}

	if ownerID == 0 {
		return nil, ErrZeroOwnerID
	}

	return m.store.Get(ctx, k, ownerID)
}

func (m *defaultManager) Delete(ctx context.Context, k Kind, ownerID int) error {
	if m.store == nil {
		return ErrNilPasswordStore
	}

	if k == 0 {
		return ErrZeroKind
	}

	if ownerID == 0 {
		return ErrZeroOwnerID
	}

	return m.store.Delete(ctx, k, ownerID)
}
