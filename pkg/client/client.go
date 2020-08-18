package client

import (
	"errors"

	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
)

const PasswordLength = 32

// Flags represent client flags
type Flags uint8

const (
	FEnabled Flags = 1 << iota
	FConfidential
	FSecret
	FEncrypted
)

// Client represents any external client that interfaces with this API
type Client struct {
	Name         bytearray.ByteString32 `db:"name" json:"name"`
	ID           uuid.UUID              `db:"id" json:"id"`
	RegisteredAt timestamp.Timestamp    `db:"registered_at" json:"registered_at"`
	ExpireAt     timestamp.Timestamp    `db:"expire_at" json:"expire_at"`
	Flags        Flags                  `db:"flags" json:"flags"`
	_            struct{}
}

func (c *Client) IsEnabled() bool      { return c.Flags&FEnabled == FEnabled }
func (c *Client) IsConfidential() bool { return c.Flags&FConfidential == FConfidential }
func (c *Client) IsExpired() bool      { return c.ExpireAt < timestamp.Now() }

func (c *Client) Validate() error {
	if c.ID == uuid.Nil {
		return ErrInvalidClientID
	}

	if c.RegisteredAt == 0 {
		return errors.New("registration timestamp is empty")
	}

	if c.Name[0] == 0 {
		return ErrNoName
	}

	return nil
}
