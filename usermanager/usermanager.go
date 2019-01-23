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
	// instance ID
	ID          ulid.ULID
	superDomain *Domain
	domains     map[ulid.ULID]*Domain
	s           Store
	sync.RWMutex
}

// New returns a new user manager instance
// also initializing by loading necessary data from a given store
func New(s Store) (*UserManager, error) {
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

	if s.ps == nil {
		return nil, ErrNilPasswordStore
	}

	// initializing the main struct
	m := &UserManager{
		domains: make(map[ulid.ULID]*Domain, 0),
		s:       s,
	}

	return m, nil
}

// Init initializes user manager
func (m *UserManager) Init() error {
	// initializing the user manager, loading domains and users from the store
	domains, err := m.s.ds.GetAll()
	if err != nil {
		return fmt.Errorf("Init(): %s", err)
	}

	// this is solely to check whether super domain exists, returning this error
	// to suggest the creation
	// NOTE: there must always be a super administrator (root) user present
	// NOTE: there must always be a super domain to which all system administrators belong
	if len(domains) == 0 {
		// TODO: check if super domain exists rather any domains, it's ok to not have any domains
		// this should trigger the creation of a super domain
		return ErrSuperDomainNotFound
	}

	// adding found domains to the dominion
	for _, d := range domains {
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

// CreateSuperDomain creates and registers a super domain only if doesn't
// already exist
func (m *UserManager) CreateSuperDomain(superuser *User) error {
	// retrieving all domains from the store to make sure there is no
	// existing super domain
	storedDomains, err := m.s.ds.GetAll()
	if err != nil {
		return err
	}

	// making sure there is no pre-existing super domain in the store
	for _, d := range storedDomains {
		if d.IsSuperDomain {
			return ErrSuperDomainExists
		}
	}

	// lookup existing super domain
	for _, d := range m.domains {
		if d.IsSuperDomain {
			return ErrSuperDomainExists
		}
	}

	// at this point it's safe to create a super domain
	sd, err := m.CreateDomain(superuser)
	if err != nil {
		return fmt.Errorf("failed to create super domain: %s", err)
	}

	// flagging as super domain
	sd.IsSuperDomain = true

	// creating the core role group for super users
	superuserRole, err := NewGroup(GKRole, "superuser", "Super User", nil)
	if err != nil {
		return fmt.Errorf("failed to create superuser group: %s", err)
	}

	//---------------------------------------------------------------------------
	// persisting changes to the store
	//---------------------------------------------------------------------------
	// storing domain
	err = sd.Save()
	if err != nil {
		return fmt.Errorf("failed to save super domain: %s", err)
	}

	// saving superuser
	err = sd.Users.Register(superuser)
	if err != nil {
		return fmt.Errorf("failed to register superuser: %s", err)
	}

	// saving the role
	err = superuserRole.Save()
	if err != nil {
		return fmt.Errorf("failed to save superuser role: %s", err)
	}

	// adding a given user to superuser role
	err = superuserRole.AddMember(superuser)
	if err != nil {
		return fmt.Errorf("failed to add user to superuser role group: %s", err)
	}

	return nil
}

// CreateDomain creating new root subdomain
func (m *UserManager) CreateDomain(owner *User) (*Domain, error) {
	// safety firstdomain must have an owner
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
	if err = m.RegisterDomain(domain); err != nil {
		return nil, fmt.Errorf("CreateDomain() failed to add domain: %s", err)
	}

	log.Printf("created new domain [%s]\n", domain.ID)

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

	// picking out the super domain
	if domain.IsSuperDomain {
		// there can be only one super domain
		if m.superDomain != nil {
			return fmt.Errorf("attempting to reassign super domain")
		}

		// assigning to the manager for ease of access
		m.superDomain = domain
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

	return nil
}
