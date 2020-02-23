package core

import (
	"context"
	"fmt"

	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/user"
	"go.uber.org/zap"
)

// userManager represents an aggregate of Hometown's core functionality
type Manager struct {
	users     *user.Manager
	groups    *user.GroupManager
	tokens    *token.Manager
	policies  *user.AccessPolicyManager
	passwords password.Manager
}

// Init initializes user manager
func (m *Manager) Init(ctx context.Context) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger().Named("[hometown]")
	l.Info("initializing core manager")

	c, err := m.Container()
	if err != nil {
		return err
	}

	//---------------------------------------------------------------------------
	// fetching all stored users and adding them to the user container
	//---------------------------------------------------------------------------
	// fetching all users
	l.Info("fetching users from the store")
	users, err := m.store.FetchUsers(ctx)
	if err != nil {
		return err
	}

	// adding found users to a user container
	l.Info("adding users to the container", zap.Int("users_found", len(users)))
	for _, u := range users {
		// initializing necessary fields after fetching
		if u.groups == nil {
			u.groups = make([]*user.Group, 0)
		}

		// adding user to container
		if err := c.Add(u); err != nil {
			// just warning and moving forward
			l.Warn(
				"Init() failed to add user to container",
				zap.Int("user_id", u.ID),
				zap.String("username", string(u.Username[:])),
				zap.Error(err),
			)
		}
	}

	//---------------------------------------------------------------------------
	// initializing group manager and injecting available users
	// to distribute them among their respective groups and roles
	//---------------------------------------------------------------------------
	l.Info("initializing group manager")
	gm, err := m.GroupManager()
	if err != nil {
		return err
	}

	err = gm.Init()
	if err != nil {
		return err
	}

	// distribute them among their respective groups
	l.Info("distributing users among their respective groups")
	err = gm.DistributeUsers(users)
	if err != nil {
		return err
	}

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
func (m *Manager) UserManager() *user.Manager {
	if m.users == nil {
		panic(ErrNilUserManager)
	}

	return m.users
}

// userManager returns a password manager object
func (m *Manager) PasswordManager() (password.Manager, error) {
	if m.passwords == nil {
		return nil, ErrNilPasswordManager
	}

	return m.passwords, nil
}

// userManager returns a group manager object
func (m *Manager) GroupManager() (*user.GroupManager, error) {
	if m.groups == nil {
		return nil, ErrNilGroupManager
	}

	return m.groups, nil
}

// userManager returns a token manager object
func (m *Manager) TokenManager() (*token.Manager, error) {
	if m.groups == nil {
		return nil, ErrNilGroupManager
	}

	return m.tokens, nil
}

// Validate validates current user manager
func (m *Manager) Validate() error {
	if m.store == nil {
		return ErrNilUserStore
	}

	if m.users == nil {
		return ErrNilUserContainer
	}

	if m.passwords == nil {
		return ErrNilPasswordManager
	}

	if m.groups == nil {
		return ErrNilGroupManager
	}

	if m.logger == nil {
		return ErrNilLogger
	}

	if m.tokens == nil {
		return ErrNilTokenManager
	}

	return nil
}

// SetLogger setting a primary logger for the core
func (m *Manager) SetLogger(logger *zap.Logger) error {
	// if logger is set, then giving it a name
	// to know the log context
	if logger != nil {
		logger = logger.Named("[USER]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise initializing and returning
// a new default emergency logger
// NOTE: will panic if it finally fails to obtain a logger
func (m *Manager) Logger() *zap.Logger {
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

func (m *Manager) setupDefaultGroups() error {
	if m.groups == nil {
		return ErrNilGroupManager
	}

	// regular user
	userRole, err := user.NewGroup(user.GKRole, "user", "Regular User", nil)
	if err != nil {
		return fmt.Errorf("failed to create regular user role: %s", err)
	}

	err = m.groups.AddGroup(userRole)
	if err != nil {
		return err
	}

	// manager
	managerRole, err := user.NewGroup(user.GKRole, "manager", "userManager", userRole)
	if err != nil {
		return fmt.Errorf("failed to create manager role: %s", err)
	}

	err = m.groups.AddGroup(managerRole)
	if err != nil {
		return err
	}

	// superuser
	superuserRole, err := user.NewGroup(user.GKRole, "superuser", "Super User", managerRole)
	if err != nil {
		return fmt.Errorf("failed to create superuser role: %s", err)
	}

	err = m.groups.AddGroup(superuserRole)
	if err != nil {
		return err
	}

	return nil
}

// SetGroupManager assigns a group manager
func (m *Manager) SetGroupManager(gm *user.GroupManager) error {
	if gm == nil {
		return ErrNilGroupManager
	}

	err := gm.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate group container: %s", err)
	}

	// setting
	m.groups = gm

	return nil
}

// SetTokenManager assigns a token manager
func (m *Manager) SetTokenManager(tm *token.Manager) error {
	if tm == nil {
		return ErrNilTokenManager
	}

	if err := tm.Validate(); err != nil {
		return fmt.Errorf("failed to validate token container: %s", err)
	}

	m.tokens = tm

	return nil
}

// SetAccessPolicyManager assigns access policy manager
func (m *Manager) SetAccessPolicyManager(c *user.AccessPolicyManager) error {
	if c == nil {
		return ErrNilAccessPolicyContainer
	}

	m.policies = c

	return nil
}
