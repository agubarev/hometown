package user

import (
	"bytes"
	"context"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// CreateUser creates a new user
func (m *Manager) CreateUser(ctx context.Context, fn func(ctx context.Context) (NewUserObject, error)) (u User, err error) {
	l := m.Logger()

	// initializing new object
	newUser, err := fn(ctx)
	if err != nil {
		return u, err
	}

	//---------------------------------------------------------------------------
	// basic cleaning and validation
	//---------------------------------------------------------------------------
	// username
	newUser.Username.Trim()
	newUser.Username.ToLower()

	// trimming password
	newUser.Password = bytes.TrimSpace(newUser.Password)

	// email
	newUser.EmailAddr.Trim()
	newUser.EmailAddr.ToLower()

	// phone
	newUser.PhoneNumber.Trim()

	// name
	newUser.Firstname.Trim()
	newUser.Middlename.Trim()
	newUser.Lastname.Trim()

	if newUser.EmailAddr[0] == 0 {
		return u, ErrEmptyEmailAddr
	}

	if len(newUser.Password) == 0 {
		return u, password.ErrEmptyPassword
	}

	//---------------------------------------------------------------------------
	// initializing and validating new user
	//---------------------------------------------------------------------------
	// initializing new user
	u = User{
		ID:        uuid.New(),
		Essential: newUser.Essential,
		Metadata: Metadata{
			CreatedAt: timestamp.Now(),
		},
	}

	// validating before storing
	if err := u.Validate(); err != nil {
		return u, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return u, err
	}

	// creating checksum
	u.Checksum = u.calculateChecksum()

	// saving to the store
	u, err = store.UpsertUser(ctx, u)
	if err != nil {
		return u, err
	}

	// deferring a function to delete this user if there's any error to follow
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(r.(error), "[panic] failed to create user with password")

			// at this point it doesn't matter what caused this panic, it only matters
			// to delete the created user and clean up what's unfinished
			// TODO: devise a contingency plan for OSHI- if the recovery fails
			if _, xerr := m.DeleteUserByID(ctx, u.ID, true); xerr != nil {
				err = errors.Wrapf(err, "[panic:critical] failed to delete user during recovery from panic: %s", xerr)
				l.Warn("failed to delete new user during recovery from panic", zap.Error(err))
			}

			//---------------------------------------------------------------------------
			// deleting things that might have been created before panic
			//---------------------------------------------------------------------------
			// deleting email
			if xerr := m.DeleteEmailByAddr(ctx, u.ID, newUser.EmailAddr); xerr != nil {
				err = errors.Wrapf(err, "failed to delete emails during recovery from panic: %s", xerr)
				l.Warn("failed to delete emails during recovery from panic", zap.Error(err))
			}

			// deleting phones
			if _, xerr := m.DeletePhoneByNumber(ctx, u.ID, newUser.PhoneNumber); xerr != nil {
				err = errors.Wrapf(err, "failed to delete phones during recovery from panic: %s", xerr)
				l.Warn("failed to delete phones during recovery from panic", zap.Error(err))
			}

			// deleting profile
			if xerr := m.DeleteProfileByUserID(ctx, u.ID); xerr != nil {
				err = errors.Wrapf(err, "failed to delete user profile during recovery from panic: %s", xerr)
				l.Warn("failed to delete user profile during recovery from panic", zap.Error(err))
			}

			// deleting password
			if xerr := m.passwords.Delete(ctx, password.OKUser, u.ID); xerr != nil {
				err = errors.Wrapf(err, "failed to delete password during recovery from panic: %s", xerr)
				l.Warn("failed to delete password during recovery from panic", zap.Error(err))
			}
		}
	}()

	//---------------------------------------------------------------------------
	// creating new email record
	//---------------------------------------------------------------------------
	_, err = m.CreateEmail(ctx, func(ctx context.Context) (object NewEmailObject, err error) {
		object = NewEmailObject{
			EmailEssential: EmailEssential{
				// passing in email address from the new user object
				Addr: newUser.EmailAddr,

				// since this email is for the new user,
				// then this is a primary email
				IsPrimary: true,
			},

			// email owner ActorID
			UserID: u.ID,

			// this email hasn't been confirmed yet
			IsConfirmed: false,
		}

		return object, nil
	})

	if err != nil {
		panic(errors.Wrapf(err, "failed to create email: %s", newUser.EmailAddr))
	}

	//---------------------------------------------------------------------------
	// creating new phone record (if number is given)
	//---------------------------------------------------------------------------
	if newUser.PhoneNumber[0] != 0 {
		_, err = m.CreatePhone(ctx, func(ctx context.Context) (object NewPhoneObject, err error) {
			object = NewPhoneObject{
				PhoneEssential: PhoneEssential{
					// passing in email address from the new user object
					Number: newUser.PhoneNumber,

					// first record for the new user means it's primary
					IsPrimary: true,
				},

				// email owner ActorID
				UserID: u.ID,

				// this phone hasn't been confirmed yet
				IsConfirmed: false,
			}

			return object, nil
		})

		if err != nil {
			panic(errors.Wrapf(err, "failed to create email: %s", newUser.EmailAddr))
		}
	}

	//---------------------------------------------------------------------------
	// creating profile
	//---------------------------------------------------------------------------
	_, err = m.CreateProfile(ctx, func(ctx context.Context) (object NewProfileObject, err error) {
		object = NewProfileObject{
			UserID:           u.ID,
			ProfileEssential: newUser.ProfileEssential,
		}

		return object, nil
	})

	if err != nil {
		panic(errors.Wrap(err, "failed to create profile"))
	}

	//---------------------------------------------------------------------------
	// creating password
	//---------------------------------------------------------------------------
	// initializing user input slice to check password safety
	userdata := []string{
		newUser.Username.String(),
		newUser.DisplayName.String(),
		newUser.Firstname.String(),
		newUser.Middlename.String(),
		newUser.Lastname.String(),
	}

	// initializing new password
	p, err := password.NewFromInput(
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize new password"))
	}

	if err = m.SetPassword(ctx, u.ID, p); err != nil {
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
		zap.String("id", u.ID.String()),
		zap.String("username", u.Username.String()),
		zap.String("email", newUser.EmailAddr.String()),
	)

	return u, nil
}

// UserByID returns a user if found by ObjectID
func (m *Manager) UserByID(ctx context.Context, id uuid.UUID) (u User, err error) {
	if id == uuid.Nil {
		return u, ErrUserNotFound
	}

	u, err = m.store.FetchUserByID(ctx, id)
	if err != nil {
		return u, errors.Wrapf(err, "failed to obtain user by id: %d", id)
	}

	return u, nil
}

// UserByUsername returns a user if found by username
func (m *Manager) UserByUsername(ctx context.Context, username bytearray.ByteString32) (u User, err error) {
	username.Trim()
	username.ToLower()

	if username[0] == 0 {
		return u, ErrUserNotFound
	}

	u, err = m.store.FetchUserByUsername(ctx, username)
	if err != nil {
		return u, errors.Wrapf(err, "failed to obtain user by username: %s", username)
	}

	return u, nil
}

// UserByEmailAddr returns a user if found by username
func (m *Manager) UserByEmailAddr(ctx context.Context, addr bytearray.ByteString256) (u User, err error) {
	addr.Trim()
	addr.ToLower()

	if addr[0] == 0 {
		return u, ErrUserNotFound
	}

	u, err = m.store.FetchUserByEmailAddr(ctx, addr)
	if err != nil {
		return u, errors.Wrapf(err, "failed to obtain user by email: %s", addr)
	}

	return u, nil
}

// UpdateUser updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdateUser(ctx context.Context, id uuid.UUID, fn func(ctx context.Context, r User) (u User, err error)) (u User, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return u, essentialChangelog, err
	}

	// obtaining existing user
	u, err = store.FetchUserByID(ctx, id)
	if err != nil {
		return u, nil, errors.Wrap(err, "failed to obtain existing user from the store")
	}

	// saving backup for further diff comparison
	backup := u

	// initializing an updated user
	updated, err := fn(ctx, backup)
	if err != nil {
		return u, nil, errors.Wrap(err, "failed to initialize updated user")
	}

	// pre-save modifications
	updated.UpdatedAt = timestamp.Now()

	/*
		// acquiring changelog of essential changes
		essentialChangelog, err = diff.Diff(u.Essential, updated.Essential)
		if err != nil {
			return u, nil, errors.Wrap(err, "failed to diff essential changes")
		}

		// acquiring total changelog
		changelog, err := diff.Diff(u, updated)
		if err != nil {
			return u, nil, errors.Wrap(err, "failed to diff total changes")
		}
	*/

	// persisting to the store as a final step
	u, err = store.UpsertUser(ctx, u)
	if err != nil {
		return u, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated user",
		zap.String("id", u.ID.String()),
		zap.String("username", u.Username.String()),
	)

	return u, essentialChangelog, nil
}

// DeleteUserByID deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeleteUserByID(ctx context.Context, id uuid.UUID, isHard bool) (u User, err error) {
	store, err := m.Store()
	if err != nil {
		return u, errors.Wrap(err, "failed to obtain a store")
	}

	if isHard {
		// hard-deleting this object
		if err = store.DeleteUserByID(ctx, id); err != nil {
			return u, errors.Wrap(err, "failed to delete user")
		}

		// and finally deleting user's password if the password manager is present
		// NOTE: it should be possible that the user could not have a password
		if m.passwords != nil {
			err = m.passwords.Delete(ctx, password.OKUser, id)
			if err != nil {
				return u, errors.Wrap(err, "failed to delete user password")
			}
		}

		return u, nil
	}

	// obtaining deleted object
	u, err = m.UserByID(ctx, id)
	if err != nil {
		return u, err
	}

	// updating to mark this object as deleted
	u, _, err = m.UpdateUser(ctx, id, func(ctx context.Context, u User) (_ User, err error) {
		u.DeletedAt = timestamp.Now()

		return u, nil
	})

	return u, nil
}

// CheckAvailability tests whether someone with such username or email is already registered
func (m *Manager) CheckAvailability(ctx context.Context, username bytearray.ByteString32, email bytearray.ByteString256) error {
	store, err := m.Store()
	if err != nil {
		return err
	}

	// TODO: check runtime cache first

	// checking storage for just in case
	_, err = store.FetchUserByUsername(ctx, username)
	if err == nil {
		return ErrUsernameTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	_, err = store.FetchUserByEmailAddr(ctx, email)
	if err == nil {
		return ErrEmailTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	return nil
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
