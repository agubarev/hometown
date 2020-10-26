package auth

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/allegro/bigcache"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Backend is an interface contract for an auth backend
type Backend interface {
	CreateSession(ctx context.Context, s *Session) (err error)
	UpdateSession(ctx context.Context, fn func(ctx context.Context, s *Session) (*Session, error)) (err error)
	DeleteSession(ctx context.Context, s *Session) (err error)
	CreateRefreshToken(ctx context.Context, rtok RefreshToken) (err error)
	UpdateRefreshToken(ctx context.Context, h RefreshTokenHash, fn func(ctx context.Context, rtok RefreshToken) (RefreshToken, error)) (rtok RefreshToken, err error)
	RotateRefreshToken(ctx context.Context, h RefreshTokenHash) (rtok RefreshToken, err error)
	DeleteRefreshToken(ctx context.Context, t RefreshToken) (err error)
	CreateAuthorizationCode(ctx context.Context, code string, challenge PKCEChallenge, tpair TokenPair) (err error)
	TokenPairByAuthorizationCode(ctx context.Context, code string) (tpair TokenPair, challenge PKCEChallenge, err error)
	DeleteAuthorizationCode(ctx context.Context, code string) (err error)
	SessionByID(ctx context.Context, jti uuid.UUID) (*Session, error)
	RefreshTokenByHash(ctx context.Context, hash RefreshTokenHash) (t RefreshToken, err error)
	ExchangeCode(ctx context.Context, code string) (tpair TokenPair, err error)
}

// DefaultBackend is a default in-memory implementation
type DefaultBackend struct {
	// a map of JTI to an actual session
	sessions map[uuid.UUID]Session

	// refresh token map { hash -> token }
	refreshTokens map[RefreshTokenHash]RefreshToken

	// a map of owner ID to a slice of session IDs
	sessionOwnership map[Identity][]uuid.UUID

	// a cache of authorization code to access tokens
	exchangeCodeCache Cache

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
		exchangeCodeCache: codeCache,
		sessions:          make(map[uuid.UUID]Session),
		refreshTokens:     make(map[RefreshTokenHash]RefreshToken),
		sessionOwnership:  make(map[Identity][]uuid.UUID),
		workerInterval:    1 * time.Minute,
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
func (b *DefaultBackend) CreateSession(ctx context.Context, s *Session) error {
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

func (b *DefaultBackend) UpdateSession(ctx context.Context, fn func(ctx context.Context, s *Session) (*Session, error)) (err error) {
	panic("implement me")
}

func (b *DefaultBackend) CreateRefreshToken(ctx context.Context, rt RefreshToken) error {
	b.Lock()
	b.refreshTokens[rt.Hash] = rt
	b.Unlock()

	return nil
}

func (b *DefaultBackend) CreateAuthorizationCode(ctx context.Context, code string, challenge PKCEChallenge, tpair TokenPair) (err error) {
	payload, err := json.Marshal(tpair)
	if err != nil {
		return errors.Wrap(err, "failed to marshal token pair")
	}

	err = b.exchangeCodeCache.Put(
		ctx,
		code,
		payload,
	)

	if err != nil {
		return errors.Wrap(err, "failed to cache authorization code and token pair")
	}

	return nil
}

func (b *DefaultBackend) ExchangeCode(ctx context.Context, code string) (tpair TokenPair, err error) {
	// obtaining cached entry
	payload, err := b.exchangeCodeCache.Get(ctx, code)
	if err != nil {
		return tpair, ErrAuthorizationCodeNotFound
	}

	if err = json.Unmarshal(payload, &tpair); err != nil {
		return tpair, errors.Wrap(err, "failed to unmarshal token pair")
	}

	// deleting cache entry
	if err = b.exchangeCodeCache.Delete(ctx, code); err != nil {
		return tpair, errors.Wrap(err, "failed to delete authorization code")
	}

	return tpair, nil
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

func (b *DefaultBackend) UpdateRefreshToken(
	ctx context.Context,
	h RefreshTokenHash,
	fn func(ctx context.Context, rtok RefreshToken) (RefreshToken, error),
) (rtok RefreshToken, err error) {
	b.Lock()
	defer b.Unlock()

	// obtaining existing token
	rtok, ok := b.refreshTokens[h]

	if !ok {
		return rtok, ErrRefreshTokenNotFound
	}

	// applying update function to this token
	rtok, err = fn(ctx, rtok)
	if err != nil {
		return rtok, errors.Wrap(err, "update function returned with an error")
	}

	// validating before stored value is replaced
	if err = rtok.Validate(); err != nil {
		return rtok, errors.Wrap(err, "updated refresh token validation has failed")
	}

	// updating stored token
	b.refreshTokens[rtok.Hash] = rtok

	return rtok, nil
}

func (b *DefaultBackend) RotateRefreshToken(ctx context.Context, h RefreshTokenHash, newToken RefreshToken) (err error) {
	b.Lock()
	defer b.Unlock()

	// obtaining existing token
	rtok, ok := b.refreshTokens[h]

	if !ok {
		return ErrRefreshTokenNotFound
	}

	// validating current token
	if err = rtok.Validate(); err != nil {
		return err
	}

	// updating stored token
	b.refreshTokens[rtok.Hash] = rtok
	b.refreshTokens[newToken.Hash] = newToken

	return nil
}

func (b *DefaultBackend) TokenPairByAuthorizationCode(ctx context.Context, code string) (tpair TokenPair, challenge PKCEChallenge, err error) {
	panic("implement me")
}

func (b *DefaultBackend) DeleteAuthorizationCode(ctx context.Context, code string) (err error) {
	panic("implement me")
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteRefreshToken(ctx context.Context, t RefreshToken) error {
	b.Lock()
	delete(b.refreshTokens, t.Hash)
	b.Unlock()

	return nil
}

// GetSessionByRefreshToken retrieves session by a refresh token
func (b *DefaultBackend) RefreshTokenByHash(ctx context.Context, hash RefreshTokenHash) (t RefreshToken, err error) {
	b.RLock()
	t, ok := b.refreshTokens[hash]
	b.RUnlock()

	if !ok {
		return t, ErrRefreshTokenNotFound
	}

	return t, nil
}
