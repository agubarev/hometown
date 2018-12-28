package usermanager

import "github.com/oklog/ulid"

// DomainStore handles domain storage
type DomainStore interface {
	PutRootDomain(d *Domain) error
	GetRootDomain() (*Domain, error)
	PutDomain(d *Domain) error
	GetAllDomains() ([]*Domain, error)
	GetDomain(id ulid.ULID) (*Domain, error)
}
