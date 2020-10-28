package client

import (
	"context"
	"sync"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type Manager struct {
	clients   map[uuid.UUID]*Client
	passwords password.Manager
	store     Store
	logger    *zap.Logger
	clock     sync.RWMutex
	ulock     sync.RWMutex
}

func NewManager(s Store) *Manager {
	return &Manager{
		clients: make(map[uuid.UUID]*Client, 0),
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

func (m *Manager) CreateClient(ctx context.Context, name string, flags Flags) (c *Client, err error) {
	c, err = New(name, flags)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize new client")
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
	if m.passwords == nil {
		return nil, ErrNilPasswordManager
	}

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

func (m *Manager) ClientByID(ctx context.Context, clientID uuid.UUID) (c *Client, err error) {
	if clientID == uuid.Nil {
		return c, ErrClientNotFound
	}

	m.clock.RLock()
	c, ok := m.clients[clientID]
	m.clock.RUnlock()

	if ok {
		return c, nil
	}

	c, err = m.store.FetchClientByID(ctx, clientID)
	if err != nil {
		return c, err
	}

	m.clock.Lock()
	m.clients[c.ID] = c
	m.clock.Unlock()

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

	m.clock.Lock()
	delete(m.clients, clientID)
	m.clock.Unlock()

	if l := m.logger; l != nil {
		l.Debug("client and password deleted", zap.String("id", clientID.String()))
	}

	return nil
}

func (m *Manager) MatchSecret(ctx context.Context, clientID uuid.UUID, rawpass []byte) (ok bool, err error) {
	if m.passwords == nil {
		return false, ErrNilPasswordManager
	}

	if clientID == uuid.Nil {
		return false, ErrClientNotFound
	}

	// obtaining password
	pass, err := m.passwords.Get(ctx, password.NewOwner(
		password.OKApplication,
		clientID,
	))

	if err != nil {
		return false, errors.Wrapf(
			err,
			"failed to obtain password for client: %s",
			clientID,
		)
	}

	return pass.Compare(rawpass), nil
}
