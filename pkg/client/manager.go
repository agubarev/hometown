package client

import (
	"context"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Manager struct {
	clients map[uuid.UUID]Client
	store   Store
}

func NewManager(s Store) *Manager {
	return &Manager{
		clients: make(map[uuid.UUID]Client, 0),
		store:   s,
	}
}

func (m *Manager) CreateClient(ctx context.Context, isConfidential bool, name bytearray.ByteString32) (c Client, err error) {
	c = Client{
		Name:         name,
		ID:           uuid.New(),
		RegisteredAt: timestamp.Now(),
		ExpireAt:     0,
		Flags:        FEnabled,
	}

	if isConfidential {
		c.Flags |= FConfidential
	}

	return c, errors.Wrap(c.Validate(), "new client has failed validation")
}

// SetPassword sets a new password for the user
func (m *Manager) SetPassword(ctx context.Context, userID uuid.UUID, p password.Password) (err error) {
	// paranoid check of whether the user is eligible to have
	// a password created and stored
	if userID == uuid.Nil {
		return errors.Wrap(ErrZeroUserID, "failed to set user password")
	}

	// storing password
	// NOTE: userManager is responsible for hashing and encryption
	if err = m.passwords.Upsert(ctx, p); err != nil {
		return errors.Wrap(err, "failed to set user password")
	}

	return nil
}
