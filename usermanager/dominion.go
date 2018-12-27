package usermanager

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/oklog/ulid"
)

// Dominion is the top level structure, a domain container
type Dominion struct {
	// root domain is a default Domain, all new fresh domains must
	// use root as a parent
	root *Domain

	// index map for faster runtime access
	idMap map[ulid.ULID]*Domain

	// default stores
	domainStore DomainStore
	userStore   UserStore
	groupStore  GroupStore
}

var (
	dominionInstance     *Dominion
	dominionInstanceOnce sync.Once
)

// InitDominion initializes the dominion singleton instance
func InitDominion(s DomainStore) error {
	if s == nil {
		return ErrNilDomainStore
	}

	// initializing dominion only once
	dominionInstanceOnce.Do(func() {
		defer glog.Flush()

		glog.Infoln("loading root domain")
		root, err := s.GetRootDomain()
		if err != nil {
			panic(fmt.Errorf("GetDominion(): %s", err))
		}

		glog.Infoln("initializing the dominion")
		// using a global package variable
		dominionInstance = &Dominion{
			root:        root,
			domainStore: s,
		}

		glog.Infoln("loading domains")
		ds, err := s.GetAllDomains()
		if err != nil {
			panic(fmt.Errorf("GetDominion(): %s", err))
		}

		for _, d := range ds {
			glog.Infof("adding domain [%s]\n", d.ID)
			err = dominionInstance.AddDomain(d)
			if err != nil {
				panic(fmt.Errorf("GetDominion(): %s", err))
			}
		}
	})

	return nil
}

// GetDominion returns the main domain structure
// function will panic() on any error
func GetDominion() *Dominion {
	if dominionInstance == nil {
		panic(fmt.Errorf("GetDominion() dominion instance is not initialized: %s", ErrNilDominion))
	}

	return dominionInstance
}

// CreateDomain creating new root subdomain
func (d *Dominion) CreateDomain(owner *User) (*Domain, error) {
	// root domain must be initialized by now
	if d.root == nil {
		return nil, ErrNilRootDomain
	}

	// domain must have an owner
	if owner == nil {
		return nil, ErrNilUser
	}

	//---------------------------------------------------------------------------
	// initializing containers
	//---------------------------------------------------------------------------
	uc, err := NewUserContainer(d.userStore)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	gc, err := NewGroupContainer(d.groupStore)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// initializing new domain
	domain, err := NewDomain(owner, d.domainStore, uc, gc)
	if err != nil {
		return nil, fmt.Errorf("CreateDomain() failed: %s", err)
	}

	// adding new domain to the root tree
	if err = d.AddDomain(domain); err != nil {
		return nil, fmt.Errorf("CreateDomain() failed to add domain: %s", err)
	}

	return domain, nil
}

// AddDomain to the dominion
func (d *Dominion) AddDomain(domain *Domain) error {
	if domain == nil {
		return ErrNilDomain
	}

	if d.root == nil {
		return ErrNilRootDomain
	}

	if _, err := d.GetDomain(domain.ID); err != ErrDomainNotFound {
		return fmt.Errorf("AddDomain(%s) failed: %s", domain, err)
	}

	glog.Infof("adding domain [%s]\n", domain.ID)

	// TODO: add subdomain

	// adding domain to ID map for faster access
	d.idMap[domain.ID] = domain

	// assigning parent domain
	// for now all domains belong to the top domain
	// TODO: add SetParent() method to Domain
	domain.Parent = d.root

	return nil
}

// GetDomain by ID
func (d *Dominion) GetDomain(id ulid.ULID) (*Domain, error) {
	if domain, ok := d.idMap[id]; ok {
		return domain, nil
	}

	return nil, ErrDomainNotFound
}
