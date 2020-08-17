package client

import (
	"context"
	"sync"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Manager struct {
	clients   map[uuid.UUID]Client
	passwords password.Manager
	store     Store
	logger    *zap.Logger
	sync.RWMutex
}

func NewManager(s Store) *Manager {
	return &Manager{
		clients: make(map[uuid.UUID]Client, 0),
		store:   s,
	}
}

func (m *Manager) SetPasswordManager(pm password.Manager) error {
	if pm == nil {
		return ErrNilPasswordManager
	}

	m.passwords = pm

	return nil
}

func (m *Manager) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named("[client]")
	}

	m.logger = logger

	return nil
}

func (m *Manager) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(errors.Wrap(err, "failed to initialize fallback logger"))
		}

		m.logger = l
	}

	return m.logger
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

	if err = c.Validate(); err != nil {
		return c, errors.Wrap(err, "new client has failed validation")
	}

	c, err = m.store.UpsertClient(ctx, c)
	if err != nil {
		return c, errors.Wrap(err, "failed to create new client")
	}

	return c, nil
}

func (m *Manager) ClientByID(ctx context.Context, clientID uuid.UUID) (c Client, err error) {
	if clientID == uuid.Nil {
		return c, ErrClientNotFound
	}

	m.RLock()
	c, ok := m.clients[clientID]
	m.RUnlock()

	if ok {
		return c, nil
	}

	c, err = m.store.FetchClientByID(ctx, clientID)
	if err != nil {
		return c, err
	}

	m.Lock()
	m.clients[c.ID] = c
	m.Unlock()

	return c, nil
}

func (m *Manager) CreatePassword(ctx context.Context, clientID uuid.UUID) (p password.Password, raw []byte, err error) {
	c, err := m.ClientByID(ctx, clientID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain client")
	}

	p, raw, err = m.passwords.Generate(ctx, 32)
	if err != nil {
		return p, raw, err
	}

	if err = m.passwords.Upsert(ctx, p); err != nil {
		return p, raw, errors.Wrap(err, "failed to set user password")
	}

	return p, raw, nil
}
