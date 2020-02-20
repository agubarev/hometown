package user

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/agubarev/hometown/pkg/password"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// CreateUser creates a new user
func (m *Manager) CreateUser(ctx context.Context, fn func(ctx context.Context) (NewUserObject, error)) (u *User, err error) {
	l := m.Logger()

	// initializing new object
	newUser, err := fn(ctx)
	if err != nil {
		return nil, err
	}

	// basic validation
	if len(newUser.Password) == 0 {
		return nil, password.ErrEmptyPassword
	}

	//---------------------------------------------------------------------------
	// attempting to initialize new password first,
	// before new user is created
	// NOTE: initializing user input slice to check password safety
	//---------------------------------------------------------------------------
	userdata := []string{
		string(newUser.Username[:]),
		string(newUser.DisplayName[:]),
		string(newUser.Firstname[:]),
		string(newUser.Middlename[:]),
		string(newUser.Lastname[:]),
	}

	// initializing new password
	p, err := password.New(newUser.Password, userdata)
	if err != nil {
		panic(err)
	}

	//---------------------------------------------------------------------------
	// initializing and validating new user
	//---------------------------------------------------------------------------
	// initializing new user
	u = &User{
		Essential: newUser.Essential,
		Metadata: Metadata{
			CreatedAt: dbr.NewNullTime(time.Now()),
		},
	}

	// validating before storing
	if err := u.Validate(); err != nil {
		return nil, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return nil, err
	}

	// creating checksum
	u.Checksum = u.calculateChecksum()

	// saving to the store
	u, err = store.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	// deferring a function to delete this user if there's any error to follow
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(r.(error), "[panic] failed to create user with password")

			// at this point it doesn't matter what caused this panic, it only matters
			// to delete the created user and clean up what's unfinished
			// TODO: devise a contingency plan for OSHI- if the recovery fails
			if _, err := m.DeleteUserByID(ctx, u.ID, true); err != nil {
				err = errors.Wrap(r.(error), "[panic:critical] failed to delete user during recovery from panic")
				l.Error("failed to delete new user during recovery from panic", zap.Error(err))
			}
		}
	}()

	if err = m.SetPassword(u, p); err != nil {
		// TODO possibly delete the unfinished user or figure out a better way to handle
		// NOTE: at this point this account cannot be allowed to stay without a password
		// because there's an explicit attempt to create a new user account WITH PASSWORD,
		// so it has to be deleted in case of an error, panic now.
		// NOTE: runtime must recover from panic, run it's contingency plan and set a proper error
		// for return
		panic(errors.Wrap(err, "failed to set password after creating new user"))
	}

	m.Logger().Debug(
		"created new user",
		zap.Int("id", u.ID),
	)

	return u, nil
}

// CreateWithPassword creates a new user with a password
func (m *Manager) CreateWithPassword(ctx context.Context, fn func(ctx context.Context) (NewUserObject, error)) (u *User, err error) {
	if m.passwords == nil {
		return nil, ErrNilPasswordManager
	}

	l := m.Logger()

	// attempting to create the user first
	u, err = m.CreateUser(ctx, fn)
	if err != nil {
		return nil, err
	}

	return
}

// BulkCreateUser creates multiple new user
func (m *Manager) BulkCreateUser(ctx context.Context, newUsers []*User) (us []*User, err error) {
	// obtaining store
	store, err := m.Store()
	if err != nil {
		return nil, err
	}

	// validating each user
	for _, u := range newUsers {
		if err = u.Validate(); err != nil {
			return nil, errors.Wrap(err, "failed to validate u before bulk creation")
		}
	}

	// saving to the store
	us, err = store.BulkCreateUser(ctx, newUsers)
	if err != nil {
		return nil, err
	}

	zap.L().Debug(
		"created a batch of users",
		zap.Int("count", len(us)),
	)

	return us, nil
}

// GetUserByID returns a user if found by ID
func (m *Manager) GetUserByID(ctx context.Context, id int) (u *User, err error) {
	if id == 0 {
		return nil, ErrUserNotFound
	}

	u, err = m.store.FetchUserByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain u")
	}

	return u, nil
}

// UpdateUser updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdateUser(ctx context.Context, id int, fn func(ctx context.Context, r User) (u User, err error)) (u *User, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return u, essentialChangelog, err
	}

	// obtaining existing user
	u, err = store.FetchUserByID(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to obtain existing u from the store")
	}

	// saving backup for further diff comparison
	backup := *u

	// initializing an updated user
	updated, err := fn(ctx, backup)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize updated u")
	}

	// pre-save modifications
	updated.UpdatedAt = dbr.NewNullTime(time.Now())

	// acquiring changelog of essential changes
	essentialChangelog, err = diff.Diff(u.Essential, updated.Essential)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff essential changes")
	}

	// acquiring total changelog
	changelog, err := diff.Diff(u, updated)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff total changes")
	}

	// persisting to the store as a final step
	u, err = store.UpdateUser(ctx, u, changelog)
	if err != nil {
		return u, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated",
		zap.Int("id", u.ID),
		zap.ByteString("username", u.Username[:]),
	)

	return u, essentialChangelog, nil
}

// DeleteUserByID deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeleteUserByID(ctx context.Context, id int, isHard bool) (u *User, err error) {
	store, err := m.Store()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain a store")
	}

	if isHard {
		// hard-deleting this object
		if err = store.DeleteUserByID(ctx, id); err != nil {
			return nil, errors.Wrap(err, "failed to delete u")
		}

		return nil, nil
	}

	// obtaining deleted object
	u, err = m.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// updating to mark this object as deleted
	u, _, err = m.UpdateUser(ctx, id, func(ctx context.Context, u User) (_ User, err error) {
		u.DeletedAt = dbr.NewNullTime(time.Now())

		return u, nil
	})

	return u, nil
}

// CheckAvailability tests whether someone with such username or email is already registered
func (m *Manager) CheckAvailability(ctx context.Context, username TUsername, email TEmailAddr) error {
	store, err := m.Store()
	if err != nil {
		return err
	}

	// runtime checks first
	_, err = store.FetchByUsername(username)
	if err == nil {
		return ErrUsernameTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	_, err = store.FetchByEmail(email)
	if err == nil {
		return ErrEmailTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	// checking storage for just in case
	_, err = store.FetchUserByKey(ctx, "username", username)
	if err == nil {
		return ErrUsernameTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	_, err = store.FetchUserByKey(ctx, "email", email)
	if err == nil {
		return ErrEmailTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	return nil
}

// UpdateAccessPolicy saves user to the store, will return an error if store is not set
func (m *Manager) Update(ctx context.Context, id int, fn func(ctx context.Context, user *User) (*User, error)) (user *User, changelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return user, changelog, err
	}

	// obtaining existing user
	user, err = store.FetchUserByID(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to update user")
	}

	// initializing an updated user
	updated, err := fn(ctx, user)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize updated user")
	}

	// pre-save modifications
	user.UpdatedAt = dbr.NewNullTime(time.Now())

	// refreshing index
	if m.index != nil {
		sid := strconv.Itoa(user.ID)

		// deleting previous index
		// TODO: workaround error cases, add logging
		err = m.index.Delete(sid)
		if err != nil {
			return user, changelog, fmt.Errorf("failed to delete previous bleve index: %s", err)
		}

		// indexing current user object version
		err = m.index.Index(sid, user)
		if err != nil {
			return user, changelog, fmt.Errorf("failed to index user object: %s", err)
		}
	}

	// acquiring changelog
	changelog, err = diff.Diff(user, updated)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff changes")
	}

	// persisting to the store as a final step
	user, err = store.Update(ctx, user)
	if err != nil {
		return user, changelog, err
	}

	m.Logger().Info(
		"updated",
		zap.Int("id", user.ID),
		zap.String("username", user.Username),
	)

	return user, changelog, nil
}

// Delete deletes user from the store and container
func (m *Manager) Delete(u *User) error {
	store, err := m.Store()
	if err != nil {
		return fmt.Errorf("Delete(): %s", err)
	}

	// NOTE: if the user doesn't exist then returning an error for
	// consistent explicitness
	_, err = m.users.GetByID(u.ID)
	if err != nil {
		return fmt.Errorf("Delete(): failed to get user %d: %s", u.ID, err)
	}

	// now deleting user from the store
	err = store.DeleteByID(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("Delete() failed to delete user from the store: %s", err)
	}

	// removing runtime object
	err = m.users.Remove(u.ID)
	if err != nil {
		return fmt.Errorf("Delete(): failed to delete user %d: %s", u.ID, err)
	}

	// and finally deleting user's password if the password manager is present
	// NOTE: it should be possible that the user could not have a password
	if m.passwords != nil {
		err = m.passwords.Delete(u)
		if err != nil {
			return fmt.Errorf("Delete(): failed to delete user password: %s", err)
		}
	}

	return nil
}

// SetPassword sets a new password for the user
func (m *Manager) SetPassword(u *User, p *password.Password) error {
	if u == nil {
		return fmt.Errorf("SetPassword(): %s", ErrNilUser)
	}

	// paranoid check of whether the user is eligible to have
	// a password created and stored
	ok, err := u.IsRegisteredAndStored()
	if err != nil {
		return fmt.Errorf("SetPassword(): %s", err)
	}

	if !ok {
		return fmt.Errorf("SetPassword(): %s", ErrUserPasswordNotEligible)
	}

	// storing password
	// NOTE: Manager is responsible for hashing and encryption
	if err = m.passwords.Create(u, p); err != nil {
		return fmt.Errorf("SetPassword(): failed to set password: %s", err)
	}

	return nil
}

// GetByName returns a user by specific key
func (m *Manager) GetByKey(keyName string, keyValue interface{}) (user *User, err error) {
	c, err := m.Container()
	if err != nil {
		return nil, err
	}

	// first, checking inside the user container
	switch keyName {
	case "id":
		user, err = c.GetByID(keyValue.(int))
	case "username":
		user, err = c.GetByUsername(keyValue.(string))
	case "email":
		user, err = c.GetByEmail(keyValue.(string))
	}

	// now, decision depending on an error value
	switch err {
	case nil:
		return user, nil
	case ErrUserNotFound:
		// obtaining user store
		s, err := m.Store()
		if err != nil {
			return nil, err
		}

		// now checking the store
		user, err = s.GetUserByKey(ctx, keyName, keyValue)
		if err != nil {
			return nil, err
		}

		// caching user by adding to the user container
		err = c.Add(user)
		if err != nil {
			return nil, err
		}

		return user, nil
	default:
		return nil, err
	}
}

// List serves as a proxy to user container's List function
func (m *Manager) List(fn func(u *User) bool) List {
	c, err := m.Container()
	if err != nil {
		// returning an empty user list (a named slice of users)
		return make(List, 0)
	}

	return c.List(fn)
}

// SetPasswordManager assigns a password manager for this container
func (m *Manager) SetPasswordManager(pm password.Manager) error {
	if pm == nil {
		return ErrNilPasswordManager
	}

	m.passwords = pm

	return nil
}

// ConfirmEmail this function is used only when user's email is confirmed
// TODO: make it one function to confirm by type (i.e. email, phone, etc.)
func (m *Manager) ConfirmEmail(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if u.EmailConfirmedAt.Valid && !u.EmailConfirmedAt.Time.IsZero() {
		return ErrUserAlreadyConfirmed
	}

	u.EmailConfirmedAt = dbr.NewNullTime(time.Now())

	if _, _, err := m.Update(ctx, 0, nil); err != nil {
		return fmt.Errorf("failed to confirm user email(%d:%s): %s", u.ID, u.Email, err)
	}

	m.Logger().Info("user email confirmed",
		zap.Int("id", u.ID),
		zap.String("email", u.Email),
	)

	return nil
}

// IsRegisteredAndStored returns true if the user is both:
// 1. registered within a user container
// 2. persisted to the store
func (m *Manager) IsRegisteredAndStored(u *User) (bool, error) {
	// checking whether the user is registered during runtime
	_, err := m.users.GetByID(u.ID)
	if err != nil {
		if err == ErrUserNotFound {
			// user isn't registered, normal return
			return false, nil
		}

		return false, err
	}

	// checking container's store
	s, err := m.Store()
	if err != nil {
		return false, err
	}

	_, err = s.GetUserByID(ctx, u.ID)
	if err != nil {
		if err == ErrUserNotFound {
			// user isn't in the store yet, normal return
			return false, nil
		}

		return false, err
	}

	return true, nil
}
