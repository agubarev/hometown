package user

import (
	"context"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
	"go.uber.org/zap"
)

// CreateProfile creates a new profile
func (m *Manager) CreateProfile(ctx context.Context, fn func(ctx context.Context) (NewProfileObject, error)) (profile Profile, err error) {
	// initializing new object
	newProfile, err := fn(ctx)
	if err != nil {
		return profile, err
	}

	// initializing new profile
	profile = Profile{
		UserID:           newProfile.UserID,
		ProfileEssential: newProfile.ProfileEssential,
		ProfileMetadata: ProfileMetadata{
			CreatedAt: dbr.NewNullTime(time.Now()),
		},
	}

	// validating profile before storing
	if err := profile.Validate(); err != nil {
		return profile, err
	}

	// obtaining store
	store, err := m.Store()
	if err != nil {
		return profile, err
	}

	// creating checksum
	profile.Checksum = profile.calculateChecksum()

	// saving to the store
	profile, err = store.UpsertProfile(ctx, profile)
	if err != nil {
		return profile, err
	}

	m.Logger().Debug(
		"created new profile",
		zap.String("id", profile.UserID.String()),
	)

	return profile, nil
}

// GetProfileByID returns a profile if found by ObjectID
func (m *Manager) GetProfileByID(ctx context.Context, id uuid.UUID) (profile Profile, err error) {
	if id == uuid.Nil {
		return profile, ErrProfileNotFound
	}

	profile, err = m.store.FetchProfileByUserID(ctx, id)
	if err != nil {
		return profile, errors.Wrap(err, "failed to obtain profile")
	}

	return profile, nil
}

// UpdateProfile updates an existing object
// NOTE: be very cautious about how you deal with metadata inside the user function
func (m *Manager) UpdateProfile(ctx context.Context, id uuid.UUID, fn func(context.Context, Profile) (Profile, error)) (profile Profile, essentialChangelog diff.Changelog, err error) {
	store, err := m.Store()
	if err != nil {
		return profile, essentialChangelog, err
	}

	// obtaining existing profile
	profile, err = store.FetchProfileByUserID(ctx, id)
	if err != nil {
		return profile, nil, errors.Wrap(err, "failed to obtain existing profile from the store")
	}

	// saving backup for further diff comparison
	backup := profile

	// initializing an updated profile
	updated, err := fn(ctx, backup)
	if err != nil {
		return profile, nil, errors.Wrap(err, "failed to initialize updated profile")
	}

	// pre-save modifications
	updated.UpdatedAt = dbr.NewNullTime(time.Now())

	/*
		// acquiring changelog of essential changes
		essentialChangelog, err = diff.Diff(profile.ProfileEssential, updated.ProfileEssential)
		if err != nil {
			return profile, nil, errors.Wrap(err, "failed to diff essential changes")
		}

		// acquiring total changelog
		changelog, err := diff.Diff(profile, updated)
		if err != nil {
			return profile, nil, errors.Wrap(err, "failed to diff total changes")
		}
	*/

	// persisting to the store as a final step
	profile, err = store.UpsertProfile(ctx, profile)
	if err != nil {
		return profile, essentialChangelog, err
	}

	m.Logger().Debug(
		"updated profile",
		zap.String("user_id", profile.UserID.String()),
	)

	return profile, essentialChangelog, nil
}

// DeleteProfileByUserID deletes an object and returns an object,
// which is an updated object if it's soft deleted, or nil otherwise
func (m *Manager) DeleteProfileByUserID(ctx context.Context, userID uuid.UUID) (err error) {
	store, err := m.Store()
	if err != nil {
		return errors.Wrap(err, "failed to obtain a store")
	}

	// hard-deleting this object
	if err = store.DeleteProfileByUserID(ctx, userID); err != nil {
		return errors.Wrapf(err, "failed to delete profile by id: %d", userID)
	}

	return nil
}
