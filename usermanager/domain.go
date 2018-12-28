package usermanager

import (
	"fmt"
	"sync"

	"github.com/golang/glog"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
type Domain struct {
	Manager *UserManager `json:"-"`

	ID           ulid.ULID       `json:"id"`
	Parent       *Domain         `json:"-"`
	Owner        *User           `json:"-"`
	Users        *UserContainer  `json:"-"`
	Groups       *GroupContainer `json:"-"`
	Subdomains   []*Domain       `json:"subdomains"`
	AccessPolicy *AccessPolicy   `json:"-"`

	store DomainStore
	sync.RWMutex
}

func (d *Domain) String() string {
	return fmt.Sprintf("domain[%s]", d.ID)
}

// NewDomain initializing new domain
func NewDomain(owner *User, s DomainStore, uc *UserContainer, gc *GroupContainer) (*Domain, error) {
	// each new domain must have an owner user assigned
	if owner == nil {
		return nil, fmt.Errorf("NewDomain() failed to set owner: %s", ErrNilUser)
	}

	// just log warning if the store isn't set at this point, for now
	if s == nil {
		glog.Warning("NewDomain: store isn't set")
	}

	// validating containers
	if err := uc.Validate(); err != nil {
		return nil, fmt.Errorf("NewDomain() failed to validate user container: %s", err)
	}

	if err := gc.Validate(); err != nil {
		return nil, fmt.Errorf("NewDomain() failed to validate group container: %s", err)
	}

	domain := &Domain{
		ID: util.NewULID(),

		// initial domain owner, full access to this domain
		Owner: owner,

		// subdomains are reserved for the future iterations because
		// I haven't thought this through, whether I want a deeper nesting
		Subdomains: make([]*Domain, 0),

		// TODO think through the default initial state of an AccessPolicy
		// at this moment each policy is independent by default
		// it doesn't have a parent by default, doesn't inherit nor extends anything
		AccessPolicy: NewAccessPolicy(owner, nil, false, false),
	}

	return domain, nil
}

// SetParent domain
func (d *Domain) SetParent(p *Domain) error {
	panic("not implemented")

	return nil
}

// Destroy this domain and everything which is safely associated with it
func (d *Domain) Destroy() error {
	panic("not implemented")

	return nil
}
