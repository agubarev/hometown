package user

import (
	"context"
	"strings"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// CreateEmail creates a new email
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
			CreatedAt: dbr.NewNullTime(time.Now()),
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
	email, err = store.CreateEmail(ctx, email)
	if err != nil {
		return email, err
	}

	m.Logger().Debug(
		"created new email",
		zap.Uint32("user_id", email.UserID),
		zap.String("addr", email.Addr),
	)

	return email, nil
}

// BulkCreateEmail creates multiple new email
func (m *Manager) BulkCreateEmail(ctx context.Context, newEmails []Email) (emails []Email, err error) {
	// obtaining store
	store, err := m.Store()
	if err != nil {
		return nil, err
	}

	// validating each email
	for _, email := range newEmails {
		if err = email.Validate(); err != nil {
			return nil, errors.Wrap(err, "failed to validate email before bulk creation")
		}
	}

	// saving to the store
	emails, err = store.BulkCreateEmail(ctx, newEmails)
	if err != nil {
		return nil, err
	}

	zap.L().Debug(
		"created a batch of emails",
		zap.Int("count", len(emails)),
	)

	return emails, nil
}

// EmailByAddr obtains an email by a given address
func (m *Manager) EmailByAddr(ctx context.Context, addr string) (email Email, err error) {
	if strings.TrimSpace(addr) == "" {
		return email, ErrEmptyEmailAddr
	}

	email, err = m.store.FetchEmailByAddr(ctx, addr)
	if err != nil {
		return email, errors.Wrapf(err, "failed to obtain email: %s", addr)
	}

	return email, nil
}

// PrimaryEmailByUserID obtains the primary email by user id
func (m *Manager) PrimaryEmailByUserID(ctx context.Context, userID uint32) (email Email, err error) {
	if userID == 0 {
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
func (m *Manager) UpdateEmail(ctx context.Context, addr string, fn func(ctx context.Context, email Email) (_ Email, err error)) (email Email, essentialChangelog diff.Changelog, err error) {
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
	updated.UpdatedAt = dbr.NewNullTime(time.Now())

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

	// persisting to the store as a final step
	email, err = store.UpdateEmail(ctx, email, changelog)
	if err != nil {
		return email, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated",
		zap.Uint32("user_id", email.UserID),
		zap.String("addr", email.Addr),
	)

	return email, essentialChangelog, nil
}

// DeleteEmailByAddr deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeleteEmailByAddr(ctx context.Context, userID uint32, addr string) (err error) {
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
func (m *Manager) ConfirmEmail(ctx context.Context, addr string) (err error) {
	if addr[0] == 0 {
		return ErrEmptyEmailAddr
	}

	email, err := m.EmailByAddr(ctx, addr)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain email by address: %s", addr)
	}

	if email.ConfirmedAt.Valid && !email.ConfirmedAt.Time.IsZero() {
		return ErrUserAlreadyConfirmed
	}

	email, _, err = m.UpdateEmail(ctx, email.Addr, func(ctx context.Context, e Email) (email Email, err error) {
		e.ConfirmedAt = dbr.NewNullTime(time.Now())

		return email, nil
	})

	if err != nil {
		return errors.Wrapf(err, "failed to confirm email: user_id=%d, addr=%s", email.UserID, email.Addr)
	}

	m.Logger().Info("email confirmed",
		zap.Uint32("user_id", email.UserID),
		zap.String("email", email.Addr),
	)

	return nil
}
