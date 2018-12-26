package usermanager

import (
	"github.com/oklog/ulid"
)

// Dominion is the top level structure, a domain container
type Dominion struct {
	root  *Domain
	idMap map[ulid.ULID]*Domain
}

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
type Domain struct {
	ID           ulid.ULID       `json:"id"`
	Parent       *Domain         `json:"-"`
	Users        *UserContainer  `json:"-"`
	Groups       *GroupContainer `json:"-"`
	AccessPolicy *AccessPolicy   `json:"accesspolicy"`
}

var dominion *Dominion

// initializing the main domain structure
func init() {
	dominion = &Dominion{}
}
