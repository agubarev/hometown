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

func (m *Manager) CreatePassword(ctx context.Context, clientID uuid.UUID) (raw []byte, err error) {
	c, err := m.ClientByID(ctx, clientID)
	if err != nil {
		return raw, errors.Wrap(err, "failed to obtain client")
	}

	o := password.Owner{
		ID:   c.ID,
		Kind: password.OKApplication,
	}

	p, raw, err := password.New(o, PasswordLength, 3, password.GFNumber|password.GFMixCase|password.GFSpecial)
	if err != nil {
		return raw, err
	}

	if err = m.passwords.Upsert(ctx, p); err != nil {
		return raw, errors.Wrap(err, "failed to set user password")
	}

	return raw, nil
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

func (m *Manager) DeleteClientByID(ctx context.Context, clientID uuid.UUID) (err error) {
	if clientID == uuid.Nil {
		return ErrClientNotFound
	}

	if err = m.store.DeleteClientByID(ctx, clientID); err != nil {
		return errors.Wrapf(err, "failed to delete client: %s", clientID)
	}

	o := password.Owner{
		ID:   clientID,
		Kind: password.OKApplication,
	}

	if err = m.passwords.Delete(ctx, o); err != nil {
		return errors.Wrapf(err, "failed to delete client password: %s", clientID)
	}

	m.Lock()
	delete(m.clients, clientID)
	m.Unlock()

	if l := m.logger; l != nil {
		l.Debug("client and password deleted", zap.String("id", clientID.String()))
	}

	return nil
}
