package user

import (
	"context"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// CreatePhone creates a new phone
func (m *Manager) CreatePhone(ctx context.Context, fn func(ctx context.Context) (NewPhoneObject, error)) (phone *Phone, err error) {
	// initializing new object
	newPhone, err := fn(ctx)
	if err != nil {
		return nil, err
	}

	// initializing newphone
	phone = &Phone{
		PhoneEssential: newPhone.PhoneEssential,
		PhoneMetadata: PhoneMetadata{
			CreatedAt: dbr.NewNullTime(time.Now()),
		},
	}

	// new phone can be confirmed
	if newPhone.IsConfirmed {
		phone.ConfirmedAt = phone.CreatedAt
	}

	// validating phone before storing
	if err := phone.Validate(); err != nil {
		return nil, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return nil, err
	}

	// saving to the store
	phone, err = store.CreatePhone(ctx, phone)
	if err != nil {
		return nil, err
	}

	m.Logger().Debug(
		"created new phone",
		zap.Int("user_id", phone.UserID),
		zap.ByteString("number", phone.Number[:]),
	)

	return phone, nil
}

// BulkCreatePhone creates multiple newphone
func (m *Manager) BulkCreatePhone(ctx context.Context, newPhones []*Phone) (phones []*Phone, err error) {
	// obtaining store
	store, err := m.Store()
	if err != nil {
		return nil, err
	}

	// validating eachphone
	for _, phone := range newPhones {
		if err = phone.Validate(); err != nil {
			return nil, errors.Wrap(err, "failed to validate phone before bulk creation")
		}
	}

	// saving to the store
	phones, err = store.BulkCreatePhone(ctx, newPhones)
	if err != nil {
		return nil, err
	}

	zap.L().Debug(
		"created a batch of phones",
		zap.Int("count", len(phones)),
	)

	return phones, nil
}

// PrimaryPhoneByUserID obtains the primary phone by user id
func (m *Manager) PrimaryPhoneByUserID(ctx context.Context, userID int) (phone *Phone, err error) {
	if userID == 0 {
		return nil, ErrPhoneNotFound
	}

	phone, err = m.store.FetchPrimaryPhoneByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain phone")
	}

	return phone, nil
}

// UpdatePhone updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdatePhone(ctx context.Context, userID int, number TPhoneNumber, fn func(ctx context.Context, e Phone) (phone Phone, err error)) (phone *Phone, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return phone, essentialChangelog, err
	}

	// obtaining existingphone
	phone, err = store.FetchPhoneByNumber(ctx, number)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to obtain existing phone from the store")
	}

	// saving backup for further diff comparison
	backup := *phone

	// initializing an updated phone
	updated, err := fn(ctx, backup)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize updated phone")
	}

	// pre-save modifications
	updated.UpdatedAt = dbr.NewNullTime(time.Now())

	// acquiring changelog of essential changes
	essentialChangelog, err = diff.Diff(phone.PhoneEssential, updated.PhoneEssential)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff essential changes")
	}

	// acquiring total changelog
	changelog, err := diff.Diff(phone, updated)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to diff total changes")
	}

	// persisting to the store as a final step
	phone, err = store.UpdatePhone(ctx, phone, changelog)
	if err != nil {
		return phone, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated",
		zap.Int("user_id", phone.UserID),
		zap.ByteString("number", phone.Number[:]),
	)

	return phone, essentialChangelog, nil
}

// DeletePhoneByNumber deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeletePhoneByNumber(ctx context.Context, userID int, number TPhoneNumber) (phone *Phone, err error) {
	store, err := m.Store()
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain a store")
	}

	// hard-deleting this object
	if err = store.DeletePhoneByNumber(ctx, userID, number); err != nil {
		return nil, errors.Wrapf(err, "failed to delete phone by number: %s", number)
	}

	return phone, nil
}
