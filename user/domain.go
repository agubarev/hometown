package user

import (
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/accesspolicy"
)

type dominion struct {
	root  *Domain
	index map[ulid.ULID]*Domain
}

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
type Domain struct {
	ID           ulid.ULID                  `json:"id"`
	Parent       *Domain                    `json:"-"`
	Name         string                     `json:"name"`
	Subdomains   []*Domain                  `json:"subdomains"`
	AccessPolicy *accesspolicy.AccessPolicy `json:"accesspolicy"`
}
