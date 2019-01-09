package usermanager

import (
	"fmt"
	"log"
	"sync"

	"github.com/oklog/ulid"
)

// UserManager is the super structure, a domain container
// this is considered a dominion, a structure that manages
// everything which is user-related
// TODO: consider naming first release `Lidia`
type UserManager struct {
	c           Config
	superDomain *Domain
	domains     map[ulid.ULID]*Domain

	sync.RWMutex
}

// Config main configuration for the user manager
// TODO: consider moving stores out of config
type Config struct {
	s Store
}

// NewConfig initializing a new user manager config
func NewConfig(s Store) Config {
	return Config{
		s: s,
	}
}

// Validate user manager config
func (c *Config) Validate() error {
	if c.s.ds == nil {
		return ErrNilDomainStore
	}

	if c.s.us == nil {
		return ErrNilUserStore
	}

	if c.s.gs == nil {
		return ErrNilGroupStore
	}

	if c.s.aps == nil {
		return ErrNilAccessPolicyStore
	}

	return nil
}

// New returns a new user manager instance
func New() *UserManager {
	// initializing the main struct
	// doing just that for now
	return &UserManager{
		domains: make(map[ulid.ULID]*Domain, 0),
	}
}

// Init initializes the user manager instance
// loads existing domains
func (m *UserManager) Init(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}

	// retrieving existing domains
	domains, err := c.s.ds.GetAll()
	if err != nil {
		return fmt.Errorf("New(): %s", err)
	}

	// preliminary checks
	// NOTE: there must always be a super administrator (root) user present
	// NOTE: there must always be a super domain to which all system administrators belong
	if len(domains) == 0 {
		return ErrEmptyDominion
	}

	// adding found domains to the dominion
	for _, d := range domains {
		err = m.AddDomain(d)
		if err != nil {
			return fmt.Errorf("New(): %s", err)
		}
	}

	return nil
}

// CreateDomain creating new root subdomain
func (m *UserManager) CreateDomain(owner *User) (*Domain, error) {
	// domain must have an owner
	if owner == nil {
		return nil, ErrNilUser
	}

	// initializing containers
	uc, err := NewUserContainer(m.c.s.us)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	gc, err := NewGroupContainer(m.c.s.gs)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// initializing new domain
	domain, err := NewDomain(owner)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// initializing new domain
	err = domain.Init(m.c.s.ds, uc, gc)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// adding new domain to the root tree
	if err = m.AddDomain(domain); err != nil {
		return nil, fmt.Errorf("CreateDomain() failed to add domain: %s", err)
	}

	log.Printf("created new domain [%s]\n", domain.ID)

	return domain, nil
}

// DestroyDomain destroy the domain and everything which is safely associated with it
func (m *UserManager) DestroyDomain(domain *Domain) error {

	return nil
}

// AddDomain existing domain to the dominion
func (m *UserManager) AddDomain(domain *Domain) error {
	if domain == nil {
		return ErrNilDomain
	}

	if _, err := m.GetDomain(domain.ID); err != ErrDomainNotFound {
		return fmt.Errorf("AddDomain(%s) failed: %s", domain, err)
	}

	log.Printf("adding domain [%s]\n", domain.ID)

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
