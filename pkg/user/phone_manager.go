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

// CreatePhone creates a new phone
func (m *Manager) CreatePhone(ctx context.Context, fn func(ctx context.Context) (NewPhoneObject, error)) (phone Phone, err error) {
	// initializing new object
	newPhone, err := fn(ctx)
	if err != nil {
		return phone, err
	}

	// initializing newphone
	phone = Phone{
		UserID:         newPhone.UserID,
		PhoneEssential: newPhone.PhoneEssential,
		PhoneMetadata: PhoneMetadata{
			CreatedAt: timestamp.Now(),
		},
	}

	// new phone can be confirmed
	if newPhone.IsConfirmed {
		phone.ConfirmedAt = phone.CreatedAt
	}

	// validating phone before storing
	if err := phone.Validate(); err != nil {
		return phone, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return phone, err
	}

	// saving to the store
	phone, err = store.UpsertPhone(ctx, phone)
	if err != nil {
		return phone, err
	}

	m.Logger().Debug(
		"created new phone",
		zap.String("user_id", phone.UserID.String()),
		zap.String("number", phone.Number.String()),
	)

	return phone, nil
}

// PrimaryPhoneByUserID obtains the primary phone by user id
func (m *Manager) PrimaryPhoneByUserID(ctx context.Context, userID uuid.UUID) (phone Phone, err error) {
	if userID == uuid.Nil {
		return phone, ErrPhoneNotFound
	}

	phone, err = m.store.FetchPrimaryPhoneByUserID(ctx, userID)
	if err != nil {
		return phone, errors.Wrap(err, "failed to obtain phone")
	}

	return phone, nil
}

// UpdatePhone updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdatePhone(ctx context.Context, number bytearray.ByteString16, fn func(ctx context.Context, phone Phone) (_ Phone, err error)) (phone Phone, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return phone, essentialChangelog, err
	}

	// obtaining existing phone
	phone, err = store.FetchPhoneByNumber(ctx, number)
	if err != nil {
		return phone, nil, errors.Wrap(err, "failed to obtain existing phone from the store")
	}

	// saving backup for further diff comparison
	backup := phone

	// initializing an updated phone
	updated, err := fn(ctx, backup)
	if err != nil {
		return phone, nil, errors.Wrap(err, "failed to initialize updated phone")
	}

	// pre-save modifications
	updated.UpdatedAt = timestamp.Now()

	/*
		// acquiring changelog of essential changes
		essentialChangelog, err = diff.Diff(phone.PhoneEssential, updated.PhoneEssential)
		if err != nil {
			return phone, nil, errors.Wrap(err, "failed to diff essential changes")
		}

		// acquiring total changelog
		changelog, err := diff.Diff(phone, updated)
		if err != nil {
			return phone, nil, errors.Wrap(err, "failed to diff total changes")
		}
	*/

	// persisting to the store as a final step
	phone, err = store.UpsertPhone(ctx, phone)
	if err != nil {
		return phone, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated phone",
		zap.String("user_id", phone.UserID.String()),
		zap.String("number", phone.Number.String()),
	)

	return phone, essentialChangelog, nil
}

// DeletePhoneByNumber deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeletePhoneByNumber(ctx context.Context, userID uuid.UUID, number bytearray.ByteString16) (phone Phone, err error) {
	store, err := m.Store()
	if err != nil {
		return phone, errors.Wrap(err, "failed to obtain a store")
	}

	// hard-deleting this object
	if err = store.DeletePhoneByNumber(ctx, userID, number); err != nil {
		return phone, errors.Wrapf(err, "failed to delete phone by number: %s", number)
	}

	return phone, nil
}
