package auth

import (
	"context"
	"log"
	"sync"
	"time"
	"unsafe"

	"github.com/allegro/bigcache"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Backend is an interface contract for an auth backend
type Backend interface {
	PutSession(ctx context.Context, s *Session) (err error)
	PutRefreshToken(ctx context.Context, rt RefreshToken) (err error)
	PutAuthCode(ctx context.Context, code string, signedToken string) (err error)
	SessionByID(ctx context.Context, jti uuid.UUID) (*Session, error)
	RefreshTokenByHash(ctx context.Context, hash Hash) (t RefreshToken, err error)
	ExchangeCode(ctx context.Context, code string) (signedToken string, err error)
	DeleteSession(ctx context.Context, s *Session) (err error)
	DeleteRefreshToken(ctx context.Context, t RefreshToken) (err error)
}

// DefaultBackend is a default in-memory implementation
type DefaultBackend struct {
	// a map of JTI to an actual session
	sessions map[uuid.UUID]Session

	// refresh token map { hash -> token }
	refreshTokens map[Hash]RefreshToken

	// a map of owner ID to a slice of session IDs
	sessionOwnership map[Identity][]uuid.UUID

	// a cache of authorization code to access tokens
	exchangeCodes Cache

	// hasWorker flags whether this backend has a cleaner worker started
	hasWorker bool

	workerInterval time.Duration
	sync.RWMutex
}

// NewDefaultRegistryBackend initializes a default in-memory registry
func NewDefaultRegistryBackend() *DefaultBackend {
	// initializing authorization code cache
	codeCache, err := NewDefaultCache(bigcache.DefaultConfig(1 * time.Minute))
	if err != nil {
		panic(errors.Wrap(
			err,
			"failed to initialize authorization code cache",
		))
	}

	b := &DefaultBackend{
		exchangeCodes:    codeCache,
		sessions:         make(map[uuid.UUID]Session),
		refreshTokens:    make(map[Hash]RefreshToken),
		sessionOwnership: make(map[Identity][]uuid.UUID),
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
			if err = b.DeleteSession(ctx, &s); err != nil {
				return errors.Wrapf(err, "failed to delete expired session: %s", s.ID)
			}

			b.Lock()
			delete(b.sessions, s.ID)
			b.Unlock()
		}
	}

	// clearing out expired sessionOwnership
	for _, sessionIDs := range b.sessionOwnership {
		for i := range sessionIDs {
			b.Lock()
			s, ok := b.sessions[sessionIDs[i]]
			b.Unlock()

			if !ok {
				// TODO delete expired session from the slice
				panic("not implemented")
			}

			if s.IsExpired() {
				if err = b.DeleteSession(ctx, &s); err != nil {
					return errors.Wrapf(err, "failed to delete expired session: %s", s.ID)
				}
			}
		}
	}

	return nil
}

// UpsertSession stores a given session to a temporary registry backend
func (b *DefaultBackend) PutSession(ctx context.Context, s *Session) error {
	if err := s.Validate(); err != nil {
		return err
	}

	b.Lock()

	// initializing a nested map and storing first value
	if b.sessionOwnership[s.Identity] == nil {
		b.sessionOwnership[s.Identity] = []uuid.UUID{s.ID}
	} else {
		// storing session until it's expired and removed
		b.sessionOwnership[s.Identity] = append(b.sessionOwnership[s.Identity], s.ID)
	}

	b.Unlock()

	return nil
}

// UpsertRefreshToken stores refresh token
func (b *DefaultBackend) PutRefreshToken(ctx context.Context, rt RefreshToken) error {
	// TODO: validation

	b.Lock()
	b.refreshTokens[rt.Hash] = rt
	b.Unlock()

	return nil
}

func (b *DefaultBackend) PutAuthCode(ctx context.Context, code string, tpair TokenPair) (err error) {
	return b.exchangeCodes.Put(
		ctx,
		code,
		tpair,
	)
}

func (b *DefaultBackend) ExchangeCode(ctx context.Context, code string) (signedToken string, err error) {
	// obtaining cached entry
	cache, err := b.exchangeCodes.Get(ctx, code)
	if err != nil {
		return "", ErrAuthorizationCodeNotFound
	}

	// converting bytes back to string
	signedToken = *(*string)(unsafe.Pointer(&cache))

	// deleting cache entry
	if err = b.exchangeCodes.Delete(ctx, code); err != nil {
		return "", errors.Wrap(err, "failed to delete authorization code")
	}

	return signedToken, nil
}

// GetSession fetches a session by a given token hash from the registry backend
func (b *DefaultBackend) SessionByID(ctx context.Context, jti uuid.UUID) (s *Session, err error) {
	if jti == uuid.Nil {
		return s, ErrInvalidJTI
	}

	// copying session as value but returning it as pointer
	b.RLock()
	ss, ok := b.sessions[jti]
	b.RUnlock()

	if !ok {
		return &ss, ErrSessionNotFound
	}

	return &ss, nil
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteSession(ctx context.Context, s *Session) error {
	if s.ID == uuid.Nil {
		return ErrInvalidSessionID
	}

	b.Lock()

	// from the main map
	delete(b.sessions, s.ID)

	// user-linked
	for i := range b.sessionOwnership[s.Identity] {
		if s.ID == b.sessionOwnership[s.Identity][i] {
			b.sessionOwnership[s.Identity] = append(
				b.sessionOwnership[s.Identity][:i],
				b.sessionOwnership[s.Identity][i+1:]...,
			)
		}
	}

	b.Unlock()

	return nil
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteRefreshToken(ctx context.Context, t RefreshToken) error {
	b.Lock()
	delete(b.refreshTokens, t.Hash)
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
