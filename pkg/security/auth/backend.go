package auth

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/agubarev/hometown/pkg/token"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Backend is an interface contract for an auth backend
type Backend interface {
	UpsertSession(ctx context.Context, s Session) (err error)
	UpsertRefreshToken(ctx context.Context, t RefreshToken) (err error)
	SessionByID(ctx context.Context, jti uuid.UUID) (Session, error)
	RefreshTokenByHash(ctx context.Context, hash Hash) (t RefreshToken, err error)
	DeleteSession(ctx context.Context, s Session) (err error)
	DeleteRefreshToken(ctx context.Context, hash token.Hash) (err error)
}

// DefaultBackend is a default in-memory implementation
type DefaultBackend struct {
	// a map of JTI to an actual session
	sessions map[uuid.UUID]Session

	// refresh token map { hash -> token }
	refreshTokens map[Hash]RefreshToken

	// a map of user ID to a slice of session IDs
	identitySessions map[Identity][]uuid.UUID

	// hasWorker flags whether this backend has a cleaner worker started
	hasWorker bool

	workerInterval time.Duration
	sync.RWMutex
}

// NewDefaultRegistryBackend initializes a default in-memory registry
func NewDefaultRegistryBackend() *DefaultBackend {
	b := &DefaultBackend{
		sessions:         make(map[uuid.UUID]Session),
		refreshTokens:    make(map[Hash]RefreshToken),
		identitySessions: make(map[Identity][]uuid.UUID),
		workerInterval:   1 * time.Minute,
	}

	// starting the maintenance worker
	if err := b.startWorker(context.TODO()); err != nil {
		panic(errors.Wrap(err, "AuthRegistryBackend: failed to start worker"))
	}

	return b
}

func (b *DefaultBackend) startWorker(ctx context.Context) error {
	if b.hasWorker {
		return errors.New("worker has already been started")
	}

	// capturing this instance by a closure
	go func() {
		log.Println("AuthRegistryBackend: worker started")

		b.hasWorker = true
		for {
			// running a blacklist cleanup
			if err := b.cleanup(ctx); err != nil {
				log.Printf("WARNING: auth registry worker has failed to cleanup: %s", err)
			}

			time.Sleep(b.workerInterval)
		}
	}()

	return nil
}

// cleanup performs registry in-memory cleanup over time
// NOTE: this is the default, but not the most optimal approach
func (b *DefaultBackend) cleanup(ctx context.Context) (err error) {
	b.Lock()
	defer b.Unlock()

	// clearing out expired items
	for _, s := range b.sessions {
		if s.IsExpired() {
			if err = b.DeleteSession(ctx, s); err != nil {
				return errors.Wrapf(err, "failed to delete expired session: %s", s.ID)
			}

			b.Lock()
			delete(b.sessions, s.ID)
			b.Unlock()
		}
	}

	// clearing out expired identitySessions
	for _, sessionIDs := range b.identitySessions {
		for i := range sessionIDs {
			b.Lock()
			s, ok := b.sessions[sessionIDs[i]]
			b.Unlock()

			if !ok {

			}

			if s.IsExpired() {
				if err = b.DeleteSession(ctx, s); err != nil {
					return errors.Wrapf(err, "failed to delete expired session: %s", s.ID)
				}
			}
		}
	}

	return nil
}

// PutSession stores a given session to a temporary registry backend
func (b *DefaultBackend) PutSession(ctx context.Context, s *Session) error {
	if err := s.Validate(); err != nil {
		return err
	}

	b.Lock()

	// initializing a nested map and storing first value
	if b.identitySessions[s.Owner] == nil {
		b.identitySessions[s.Owner] = []uuid.UUID{s.ID}
	} else {
		// storing session until it's expired and removed
		b.identitySessions[s.Owner] = append(b.identitySessions[s.Owner], s.ID)
	}

	b.Unlock()

	return nil
}

// GetSession fetches a session by a given token hash from the registry backend
func (b *DefaultBackend) SessionByID(ctx context.Context, jti uuid.UUID) (s Session, err error) {
	if jti == uuid.Nil {
		return s, ErrInvalidJTI
	}

	b.RLock()
	s, ok := b.sessions[jti]
	b.RUnlock()

	if !ok {
		return s, ErrSessionNotFound
	}

	return s, nil
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteSession(ctx context.Context, s Session) error {
	if s.ID == uuid.Nil {
		return ErrInvalidSessionID
	}

	b.Lock()

	// from the main map
	delete(b.sessions, s.ID)

	// user-linked
	for i := range b.identitySessions[s.Owner] {
		if s.ID == b.identitySessions[s.Owner][i] {
			b.identitySessions[s.Owner] = append(
				b.identitySessions[s.Owner][:i],
				b.identitySessions[s.Owner][i+1:]...,
			)
		}
	}

	b.Unlock()

	return nil
}

// GetSessionByRefreshToken retrieves session by a refresh token
func (b *DefaultBackend) RefreshTokenByHash(ctx context.Context, hash Hash) (t RefreshToken, err error) {
	b.RLock()
	t, ok := b.refreshTokens[hash]
	b.RUnlock()

	if !ok {
		return t, ErrRefreshTokenNotFound
	}

	return t, nil
}
