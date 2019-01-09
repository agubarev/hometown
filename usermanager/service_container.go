package usermanager

import (
	"github.com/oklog/ulid"
)

// ConsumerService represents a consumer service that has the permission
// to address the user manager service
type ConsumerService struct {
	ID   ulid.ULID `json:"id"`
	Name string    `json:"name"`
}

// ServiceContainer manages service consumers access to this API
type ServiceContainer struct {
}
