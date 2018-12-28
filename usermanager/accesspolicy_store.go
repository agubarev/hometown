package usermanager

import "github.com/oklog/ulid"

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
type AccessPolicyStore interface {
	Put(p *AccessPolicy)
	GetByID(id ulid.ULID) *AccessPolicy
	GetByUser(u *User) []*AccessPolicy
	GetByNamespace(ns string) []*AccessPolicy
	GetAll() []*AccessPolicy
}
