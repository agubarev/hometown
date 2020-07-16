package user

import (
	"fmt"
	"sync"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ContextKey uint16

const (
	CKUserManager ContextKey = iota
	CKGroupManager
	CKAccessPolicyManager
)

// Member represents a group member contract
type Object interface {
	ObjectID() int64
	ObjectKind() uint8
}

// userManager handles business logic of its underlying objects
// TODO: consider naming first release `Lidia`
type Manager struct {
	passwords password.Manager
	groups    *group.Manager
	policies  *accesspolicy.Manager
	tokens    *token.Manager
	store     Store
	logger    *zap.Logger
	sync.RWMutex
}

// NewManager returns a new user manager instance
// also initializing by loading necessary data from a given store
func NewManager(s Store) (*Manager, error) {
	if s == nil {
		return nil, errors.Wrap(ErrNilUserStore, "failed to initialize user manager")
	}

	// initializing the user manager
	m := &Manager{
		store: s,
	}

	// using default logger
	logger, err := util.DefaultLogger(true, "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default logger: %s", err)
	}

	err = m.SetLogger(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to set default logger: %s", err)
	}

	return m, nil
}

func (m *Manager) Validate() error {
	if m.passwords == nil {
		return ErrNilPasswordManager
	}

	if m.store == nil {
		return ErrNilUserStore
	}

	return nil
}

// Store returns store if set
func (m *Manager) Store() (Store, error) {
	if m.store == nil {
		return nil, ErrNilUserStore
	}

	return m.store, nil
}

// SetLogger assigns a primary logger for the manager
func (m *Manager) SetLogger(logger *zap.Logger) error {
	// if logger is set, then giving it a name
	// to know the log context
	if logger != nil {
		logger = logger.Named("[user]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise
// initializing and returning a new default emergency logger
// NOTE: will panic if it finally fails to obtain a logger
func (m *Manager) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			// having a working logger is crucial, thus must panic() if initialization fails
			panic(errors.Wrap(err, "failed to initialize fallback logger"))
		}

		m.logger = l
	}

	return m.logger
}

// SetPasswordManager assigns a password manager for this container
func (m *Manager) SetPasswordManager(pm password.Manager) error {
	if pm == nil {
		return ErrNilPasswordManager
	}

	m.passwords = pm

	return nil
}

// SetGroupManager assigns a group manager
func (m *Manager) SetGroupManager(gm *group.Manager) error {
	if gm == nil {
		return group.ErrNilManager
	}

	// setting
	m.groups = gm

	return nil
}

// SetTokenManager assigns a token manager
func (m *Manager) SetTokenManager(tm *token.Manager) error {
	if tm == nil {
		return token.ErrNilTokenManager
	}

	if err := tm.Validate(); err != nil {
		return fmt.Errorf("failed to validate token container: %s", err)
	}

	m.tokens = tm

	return nil
}

// SetAccessPolicyManager assigns access policy manager
func (m *Manager) SetAccessPolicyManager(apm *accesspolicy.Manager) error {
	if apm == nil {
		return accesspolicy.ErrNilAccessPolicyManager
	}

	m.policies = apm

	return nil
}

func (m *Manager) GroupManager() *group.Manager {
	if m.groups == nil {
		panic(group.ErrNilManager)
	}

	return m.groups
}

func (m *Manager) AccessPolicyManager() *accesspolicy.Manager {
	if m.passwords == nil {
		panic(accesspolicy.ErrNilAccessPolicyManager)
	}

	return m.policies
}

func (m *Manager) TokenManager() *token.Manager {
	if m.tokens == nil {
		panic(token.ErrNilTokenManager)
	}

	return m.tokens
}

func (m *Manager) PasswordManager() password.Manager {
	if m.passwords == nil {
		panic(ErrNilPasswordManager)
	}

	return m.passwords
}
