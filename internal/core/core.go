package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/user"
	"go.uber.org/zap"
)

// errors
var (
	ErrNilCore = errors.New("hometown core is nil")
)

type Core struct {
	users  *user.Manager
	tokens *token.Manager
	logger *zap.Logger
}

// Init initializes user manager
func (m *Core) Init(ctx context.Context) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger().Named("[hometown]")
	l.Info("initializing the core")

	// TODO: ...

	//---------------------------------------------------------------------------
	// initializing token manager
	//---------------------------------------------------------------------------
	l.Info("initializing token manager")
	tm, err := m.TokenManager()
	if err != nil {
		return err
	}

	err = tm.Init()
	if err != nil {
		return err
	}

	return nil
}

// UserManager returns a user manager object
func (m *Core) UserManager() *user.Manager {
	if m.users == nil {
		panic(user.ErrNilManager)
	}

	return m.users
}

// userManager returns a token manager object
func (m *Core) TokenManager() (*token.Manager, error) {
	if m.tokens == nil {
		return nil, token.ErrNilTokenManager
	}

	return m.tokens, nil
}

// SanitizeAndValidate validates current user manager
func (m *Core) Validate() error {
	if m.users == nil {
		return user.ErrNilManager
	}

	if m.tokens == nil {
		return token.ErrNilTokenManager
	}

	return nil
}

// SetLogger setting a primary logger for the core
func (m *Core) SetLogger(logger *zap.Logger) error {
	// if logger is set, then giving it a name
	// to know the log context
	if logger != nil {
		logger = logger.Named("[hometown]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise initializing and returning
// a new default emergency logger
// NOTE: will panic if it finally fails to obtain a logger
func (m *Core) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			// having a working logger is crucial, thus must panic() if initialization fails
			panic(fmt.Errorf("failed to initialize core logger: %s", err))
		}

		m.logger = l
	}

	return m.logger
}

// SetTokenManager assigns a token manager
func (m *Core) SetTokenManager(tm *token.Manager) error {
	if tm == nil {
		return token.ErrNilTokenManager
	}

	if err := tm.Validate(); err != nil {
		return fmt.Errorf("failed to validate token manager: %s", err)
	}

	m.tokens = tm

	return nil
}
