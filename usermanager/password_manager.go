package usermanager

// PasswordManager describes the behaviour of a user password manager
type PasswordManager interface {
	Set(u *User, p Password) error
	Get(u *User) (*Password, error)
	Delete(u *User) error
}

type defaultPasswordManager struct {
	ps PasswordStore
}

// NewDefaultPasswordManager initializes the default user password manager
func NewDefaultPasswordManager(ps PasswordStore) (PasswordManager, error) {
	if ps == nil {
		return nil, ErrNilPasswordStore
	}

	pm := &defaultPasswordManager{
		ps: ps,
	}

	return pm, nil
}

func (pm *defaultPasswordManager) Set(u *User, p Password) error {
	if pm.ps == nil {
		return ErrNilPasswordStore
	}

	if u == nil {
		return ErrNilUser
	}

	return pm.ps.Put(u.ID, p.Hash)
}

func (pm *defaultPasswordManager) Get(u *User) (*Password, error) {
	if pm.ps == nil {
		return nil, ErrNilPasswordStore
	}

	if u == nil {
		return nil, ErrNilUser
	}

	pbytes, err := pm.ps.Get(u.ID)
	if err != nil {
		return nil, err
	}

	p := &Password{
		ID:   u.ID,
		Hash: pbytes,
	}

	return p, nil
}

func (pm *defaultPasswordManager) Delete(u *User) error {
	if pm.ps == nil {
		return ErrNilPasswordStore
	}

	if u == nil {
		return ErrNilUser
	}

	return pm.ps.Delete(u.ID)
}
