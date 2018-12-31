package usermanager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/oklog/ulid"
)

// UserManager is the top level structure, a domain container
type UserManager struct {
	// rootDomain domain is a default Domain, all new fresh domains must
	// use rootDomain as a parent
	rootDomain *Domain

	// domain index map for faster runtime access
	idMap map[ulid.ULID]*Domain

	c Config
	sync.RWMutex
}

// Config main configuration for the user manager
type Config struct {
	ctx context.Context
	s   Store
}

// NewConfig initializing a new user manager config
func NewConfig(ctx context.Context, s Store) Config {
	return Config{
		ctx: ctx,
		s:   s,
	}
}

// Validate user manager config
func (c *Config) Validate() error {
	if c.ctx == nil {
		return ErrNilRootContext
	}

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

// NewUserManager initializing a new user manager
func NewUserManager(c Config) (*UserManager, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	root, err := c.s.ds.GetRoot(c.ctx)
	if err != nil {
		panic(fmt.Errorf("NewUserManager(): %s", err))
	}

	// initializing the main struct
	manager := &UserManager{
		rootDomain: root,
		c:          c,
	}

	domains, err := c.s.ds.GetAll(c.ctx)
	if err != nil {
		panic(fmt.Errorf("NewUserManager(): %s", err))
	}

	for _, d := range domains {
		err = manager.AddDomain(d)
		if err != nil {
			panic(fmt.Errorf("NewUserManager(): %s", err))
		}
	}

	return manager, nil
}

// CreateDomain creating new root subdomain
func (manager *UserManager) CreateDomain(owner *User) (*Domain, error) {
	// root domain must be initialized by now
	if manager.rootDomain == nil {
		return nil, ErrNilRootDomain
	}

	// domain must have an owner
	if owner == nil {
		return nil, ErrNilUser
	}

	// initializing containers
	uc, err := NewUserContainer(manager.c.s.us)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	gc, err := NewGroupContainer(manager.c.s.gs)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// initializing new domain
	domain, err := NewDomain(owner, manager.c.s.ds, uc, gc)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// adding new domain to the root tree
	if err = manager.AddDomain(domain); err != nil {
		return nil, fmt.Errorf("CreateDomain() failed to add domain: %s", err)
	}

	log.Printf("created new domain [%s]\n", domain.ID)

	return domain, nil
}

// AddDomain to the dominion
func (manager *UserManager) AddDomain(domain *Domain) error {
	if domain == nil {
		return ErrNilDomain
	}

	if manager.rootDomain == nil {
		return ErrNilRootDomain
	}

	if _, err := manager.GetDomain(domain.ID); err != ErrDomainNotFound {
		return fmt.Errorf("AddDomain(%s) failed: %s", domain, err)
	}

	log.Printf("adding domain [%s]\n", domain.ID)

	// TODO: add subdomain

	// adding domain to ID map for faster access
	manager.Lock()
	manager.idMap[domain.ID] = domain

	// assigning parent domain
	// for now all domains belong to the top domain
	// TODO: add SetParent() method to Domain
	domain.Parent = manager.rootDomain
	manager.Unlock()

	return nil
}

// GetDomain by ID
func (manager *UserManager) GetDomain(id ulid.ULID) (*Domain, error) {
	manager.RLock()
	defer manager.RUnlock()

	if domain, ok := manager.idMap[id]; ok {
		return domain, nil
	}

	return nil, ErrDomainNotFound
}
