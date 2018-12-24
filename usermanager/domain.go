package user

import (
	"github.com/oklog/ulid"
)

type dominion struct {
	root  *Domain
	idMap map[ulid.ULID]*Domain
}

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
type Domain struct {
	ID           ulid.ULID       `json:"id"`
	Parent       *Domain         `json:"-"`
	Name         string          `json:"name"`
	Groups       *GroupContainer `json:"-"`
	Subdomains   []*Domain       `json:"subdomains"`
	AccessPolicy *AccessPolicy   `json:"accesspolicy"`
}
