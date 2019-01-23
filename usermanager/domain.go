package usermanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
// IMPORTANT: any operations related to a protected super domain require
// extreme caution because it contains all system level user administrators
type Domain struct {
	ID            ulid.ULID       `json:"id"`
	Name          string          `json:"n"`
	Owner         *User           `json:"-"`
	Users         *UserContainer  `json:"-"`
	Groups        *GroupContainer `json:"-"`
	AccessPolicy  *AccessPolicy   `json:"-"`
	IsSuperDomain bool            `json:"isd"`

	CreatedAt   time.Time `json:"t_cr"`
	UpdatedAt   time.Time `json:"t_up,omitempty"`
	ConfirmedAt time.Time `json:"t_co,omitempty"`

	store DomainStore
	sync.RWMutex
}

// IDString is a short text info
func (d *Domain) IDString() string {
	return fmt.Sprintf("domain[%s]", d.ID)
}

// NewDomain initializing new domain
func NewDomain(owner *User) (*Domain, error) {
	// each new domain must have an owner user assigned
	if owner == nil {
		return nil, fmt.Errorf("NewDomain() failed to set owner: %s", ErrNilUser)
	}

	domain := &Domain{
		ID: util.NewULID(),

		// initial domain owner, full access to this domain
		Owner: owner,

		// at this moment each policy is independent by default
		// it doesn't have a parent by default, doesn't inherit nor extends anything
		// TODO think through the default initial state of an AccessPolicy
		// TODO: add domain ID as policy ID
		AccessPolicy: NewAccessPolicy(owner, nil, false, false),
	}

	return domain, nil
}

// Init the domain
func (d *Domain) Init(s DomainStore, uc *UserContainer, gc *GroupContainer) error {
	// domain store must be set during initialization
	if s == nil {
		return ErrNilDomainStore
	}

	// validating containers
	if err := uc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate user container: %s", err)
	}

	if err := gc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate group container: %s", err)
	}

	return nil
}

// Save stores the domain object
func (d *Domain) Save() error {
	if d.store == nil {
		return ErrNilDomainStore
	}

	if err := d.store.Put(d); err != nil {
		return fmt.Errorf("failed to store a domain: %s", err)
	}

	return nil
}
