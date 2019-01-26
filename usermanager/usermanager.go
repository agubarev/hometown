package usermanager

import (
	"fmt"
	"sync"

	"github.com/oklog/ulid"
)

// UserManager is the super structure, a domain container
// this is considered a dominion, a structure that manages
// everything which is user-related
// TODO: consider naming first release `Lidia`
type UserManager struct {
	// instance ID
	ID      ulid.ULID
	domains map[ulid.ULID]*Domain
	sync.RWMutex
}

// New returns a new user manager instance
// also initializing by loading necessary data from a given store
// TODO: add usermanager config to initialization
func New() (*UserManager, error) {
	// initializing the main struct
	m := &UserManager{
		domains: make(map[ulid.ULID]*Domain, 0),
	}

	return m, nil
}

// Init initializes user manager
func (m *UserManager) Init() error {
	// initializing the user manager, loading domains and users from the store
	ds, err := m.s.ds.GetAll()
	if err != nil {
		return fmt.Errorf("Init(): %s", err)
	}

	// adding found domains to the user manager
	for _, d := range ds {
		// checking whether its already registered
		if _, err := m.GetDomain(d.ID); err != ErrDomainNotFound {
			return ErrDuplicateDomain
		}

		err = m.RegisterDomain(d)
		if err != nil {
			return fmt.Errorf("Init(): %s", err)
		}
	}

	return nil
}

// CreateDomain creating new root subdomain
func (m *UserManager) CreateDomain(owner *User) (*Domain, error) {
	// safety firstdomain must have an owner
	if owner == nil {
		return nil, ErrNilUser
	}

	// initializing new domain
	domain, err := NewDomain(owner)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// initializing new domain
	err = domain.Init(m.s.ds, uc, gc)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// adding new domain to the root tree
	if err = m.RegisterDomain(domain); err != nil {
		return nil, fmt.Errorf("CreateDomain() failed to add domain: %s", err)
	}

	return domain, nil
}

// DestroyDomain destroy the domain and everything which is safely associated with it
func (m *UserManager) DestroyDomain(domain *Domain) error {

	return nil
}

// RegisterDomain existing domain to the dominion
func (m *UserManager) RegisterDomain(domain *Domain) error {
	if domain == nil {
		return ErrNilDomain
	}

	if _, err := m.GetDomain(domain.ID); err != ErrDomainNotFound {
		return fmt.Errorf("AddDomain(%s) failed: %s", domain.IDString(), err)
	}

	// adding domain to ID map for faster access
	m.Lock()
	m.domains[domain.ID] = domain
	m.Unlock()

	return nil
}

// GetDomain by ID
func (m *UserManager) GetDomain(id ulid.ULID) (*Domain, error) {
	m.RLock()
	defer m.RUnlock()

	if domain, ok := m.domains[id]; ok {
		return domain, nil
	}

	return nil, ErrDomainNotFound
}

// SetUserPassword sets a new password for the user
func (m *UserManager) SetUserPassword(user *User, newpass string) error {
	if user == nil {
		return ErrNilUser
	}

	if m.s.ps == nil {
		return ErrNilPasswordStore
	}

	// TODO: implement
	panic("not implemented")

	return nil
}
