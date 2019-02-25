package usermanager

import (
	"fmt"
	"io/ioutil"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// UserManager is the super structure, a domain container
// this is considered a dominion, a structure that manages
// everything which is user-related
// TODO: consider naming first release `Lidia`
type UserManager struct {
	// instance ID
	ID ulid.ULID

	domains map[ulid.ULID]*Domain
	sync.RWMutex
}

// New returns a new user manager instance
// also initializing by loading necessary data from a given store
// TODO: add usermanager config to initialization
func New() (*UserManager, error) {
	// initializing the main struct
	m := &UserManager{
		ID:      util.NewULID(),
		domains: make(map[ulid.ULID]*Domain, 0),
	}

	return m, nil
}

// Init initializes user manager
func (m *UserManager) Init() error {
	// initializing the user manager, loading domains and users from the storage
	ds, err := m.GetDomains()
	if err != nil {
		return fmt.Errorf("failed to load domains: %s", err)
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

	// using default config for now
	dopts, err := DefaultDomainOptions()

	// initializing new domain
	domain, err := NewDomain(owner, dopts)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement

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

// RegisterDomain add existing domain to the dominion
func (m *UserManager) RegisterDomain(d *Domain) error {
	if d == nil {
		return ErrNilDomain
	}

	// test whether the domain is already registered by attempting
	// to obtain it from the map
	if _, err := m.GetDomain(d.ID); err != ErrDomainNotFound {
		return fmt.Errorf("AddDomain(%s) failed: %s", d.StringID(), err)
	}

	// adding domain to the map
	m.Lock()
	m.domains[d.ID] = d
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

// GetDomains returns the list of domains registered within this instance
func (m *UserManager) GetDomains() ([]*Domain, error) {
	ds := make([]*Domain, 0)

	err := validateLocalStorageDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain domain list: %s", err)
	}

	// obtaining file listing
	fs, err := ioutil.ReadDir(localStorageDirectory())
	if err != nil {
		return nil, fmt.Errorf("failed to read local storage directory %s: %s", localStorageDirectory(), err)
	}

	// iterating over files and considering each directory file as domain
	for _, f := range fs {
		if !f.IsDir() {
			// just printing a warning
			fmt.Printf("WARNING: non-directory file inside local storage directory [%s]", f.Name())
			continue
		}

		// attempting to parse filename as id
		id, err := ulid.ParseStrict(f.Name())
		if err != nil {
			return nil, err
		}

		// attempting to load this directory as domain
		d, err := LoadDomain(id)
		if err != nil {
			return nil, fmt.Errorf("failed to load domain %s: %s", id, err)
		}

		// appending domain to resulting slice
		ds = append(ds, d)
	}

	return ds, nil
}
