package usermanager

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"

	"go.etcd.io/bbolt"

	"github.com/oklog/ulid"
)

// Config is a structure stored inside the main datafile
// it contains settings that wouldn't be go inside a configuration file
type Config struct {
	// TODO: do I really need this?
}

// UserManager is the super structure, a domain container
// this is considered a dominion, a structure that manages
// everything which is user-related
// TODO: consider naming first release `Lidia`
type UserManager struct {
	superDomain *Domain
	domains     map[ulid.ULID]*Domain

	s Store
	sync.RWMutex
	db *bbolt.DB
}

// New returns a new user manager instance
// also initializing by loading necessary data from a given store
func New(s Store) (*UserManager, error) {
	// initializing the main struct
	m := &UserManager{
		domains: make(map[ulid.ULID]*Domain, 0),
	}

	//---------------------------------------------------------------------------
	// validating the store
	//---------------------------------------------------------------------------
	if s.ds == nil {
		return nil, ErrNilDomainStore
	}

	if s.us == nil {
		return nil, ErrNilUserStore
	}

	if s.gs == nil {
		return nil, ErrNilGroupStore
	}

	if s.aps == nil {
		return nil, ErrNilAccessPolicyStore
	}

	//---------------------------------------------------------------------------
	// initializing the user manager
	// loading domains and users from the store
	//---------------------------------------------------------------------------
	// initializing manager specific database
	if viper.GetString("instance.datafile") == "" {
		return nil, ErrDatafileNotSpecified
	}

	db, err := bbolt.Open(viper.GetString("instance.datafile"), 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("New(): failed to open datafile: %s", err)
	}

	// loading super domain

	// retrieving existing domains
	domains, err := s.ds.GetAll()
	if err != nil {
		return nil, fmt.Errorf("New(): %s", err)
	}

	// preliminary checks
	// NOTE: there must always be a super administrator (root) user present
	// NOTE: there must always be a super domain to which all system administrators belong
	if len(domains) == 0 {
		return nil, ErrEmptyDominion
	}

	// adding found domains to the dominion
	for _, d := range domains {
		err = m.AddDomain(d)
		if err != nil {
			return nil, fmt.Errorf("New(): %s", err)
		}
	}

	return m, nil
}

// CreateDomain creating new root subdomain
func (m *UserManager) CreateDomain(owner *User) (*Domain, error) {
	// domain must have an owner
	if owner == nil {
		return nil, ErrNilUser
	}

	// initializing containers
	uc, err := NewUserContainer(m.s.us)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	gc, err := NewGroupContainer(m.s.gs)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
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
