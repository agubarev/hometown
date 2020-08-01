package user

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// UpsertEmail creates a new email
func (m *Manager) CreateEmail(ctx context.Context, fn func(ctx context.Context) (NewEmailObject, error)) (email Email, err error) {
	// initializing new object
	newEmail, err := fn(ctx)
	if err != nil {
		return email, err
	}

	// initializing new email
	email = Email{
		UserID:         newEmail.UserID,
		EmailEssential: newEmail.EmailEssential,
		EmailMetadata: EmailMetadata{
			CreatedAt: timestamp.Now(),
		},
	}

	// new email can be confirmed
	if newEmail.IsConfirmed {
		email.ConfirmedAt = email.CreatedAt
	}

	// validating email before storing
	if err := email.Validate(); err != nil {
		return email, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return email, err
	}

	// saving to the store
	email, err = store.UpsertEmail(ctx, email)
	if err != nil {
		return email, err
	}

	m.Logger().Debug(
		"created new email",
		zap.String("user_id", email.UserID.String()),
		zap.String("addr", email.Addr.String()),
	)

	return email, nil
}

// EmailByAddr obtains an email by a given address
func (m *Manager) EmailByAddr(ctx context.Context, addr bytearray.ByteString256) (email Email, err error) {
	addr.Trim()
	addr.ToLower()

	if addr[0] == 0 {
		return email, ErrEmptyEmailAddr
	}

	email, err = m.store.FetchEmailByAddr(ctx, addr)
	if err != nil {
		return email, errors.Wrapf(err, "failed to obtain email: %s", addr)
	}

	return email, nil
}

// PrimaryEmailByUserID obtains the primary email by user id
func (m *Manager) PrimaryEmailByUserID(ctx context.Context, userID uuid.UUID) (email Email, err error) {
	if userID == uuid.Nil {
		return email, ErrEmailNotFound
	}

	email, err = m.store.FetchPrimaryEmailByUserID(ctx, userID)
	if err != nil {
		return email, errors.Wrap(err, "failed to obtain email")
	}

	return email, nil
}

// UpdateEmail updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdateEmail(ctx context.Context, addr bytearray.ByteString256, fn func(ctx context.Context, email Email) (_ Email, err error)) (email Email, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return email, essentialChangelog, err
	}

	// obtaining existing email
	email, err = store.FetchEmailByAddr(ctx, addr)
	if err != nil {
		return email, nil, errors.Wrap(err, "failed to obtain existing email from the store")
	}

	// saving backup for further diff comparison
	backup := email

	// initializing an updated email
	updated, err := fn(ctx, backup)
	if err != nil {
		return email, nil, errors.Wrap(err, "failed to initialize updated email")
	}

	// pre-save modifications
	updated.UpdatedAt = timestamp.Now()

	/*
		// acquiring changelog of essential changes
		essentialChangelog, err = diff.Diff(email.EmailEssential, updated.EmailEssential)
		if err != nil {
			return email, nil, errors.Wrap(err, "failed to diff essential changes")
		}

			// acquiring total changelog
			changelog, err := diff.Diff(email, updated)
			if err != nil {
				return email, nil, errors.Wrap(err, "failed to diff total changes")
			}
	*/

	// persisting to the store as a final step
	email, err = store.UpsertEmail(ctx, email)
	if err != nil {
		return email, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated email",
		zap.String("user_id", email.UserID.String()),
		zap.String("addr", email.Addr.String()),
	)

	return email, essentialChangelog, nil
}

// DeleteEmailByAddr deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeleteEmailByAddr(ctx context.Context, userID uuid.UUID, addr bytearray.ByteString256) (err error) {
	store, err := m.Store()
	if err != nil {
		return errors.Wrap(err, "failed to obtain a store")
	}

	// hard-deleting this object
	if err = store.DeleteEmailByAddr(ctx, userID, addr); err != nil {
		return errors.Wrapf(err, "failed to delete email: %s", addr)
	}

	return nil
}

// ConfirmEmail this function is used only when user's email is confirmed
func (m *Manager) ConfirmEmail(ctx context.Context, addr bytearray.ByteString256) (err error) {
	if addr[0] == 0 {
		return ErrEmptyEmailAddr
	}

	email, err := m.EmailByAddr(ctx, addr)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain email by address: %s", addr)
	}

	if email.ConfirmedAt != 0 {
		return ErrUserAlreadyConfirmed
	}

	email, _, err = m.UpdateEmail(ctx, email.Addr, func(ctx context.Context, e Email) (email Email, err error) {
		e.ConfirmedAt = timestamp.Now()

		return email, nil
	})

	if err != nil {
		return errors.Wrapf(err, "failed to confirm email: addr=%s", addr)
	}

	m.Logger().Info("email confirmed",
		zap.String("user_id", email.UserID.String()),
		zap.String("email", email.Addr.String()),
	)

	return nil
}
