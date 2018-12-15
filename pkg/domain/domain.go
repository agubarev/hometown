package domain

import (
	"github.com/oklog/ulid"
	ap "gitlab.com/agubarev/hometown/pkg/accesspolicy"
)

type dominion struct {
	root  *Domain
	index map[ulid.ULID]*Domain
	store Store
}

// Domain represents a single organizational unit
type Domain struct {
	ID         ulid.ULID        `json:"id"`
	Parent     *Domain          `json:"-"`
	Name       string           `json:"name"`
	Subdomains []Domain         `json:"subdomains"`
	Policy     *ap.AccessPolicy `json:"policy"`
}
