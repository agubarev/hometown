package auth

import (
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
)

ClientSecret

// Client represents any external client that interfaces with this API
type Client struct {
	ID     uuid.UUID              `db:"id" json:"id"`
	Name   bytearray.ByteString32 `db:"name" json:"name"`
	Secret Secret                 `db:"secret" json:"secret"`
}
