package auth

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Backend is an interface contract for an auth backend
type Backend interface {
	UpsertSession(ctx context.Context, s Session) (err error)
	UpsertRefreshToken(ctx context.Context, refreshToken RefreshToken) (err error)
	SessionByID(ctx context.Context, hash token.Hash) (Session, error)
	SessionByRefreshToken(ctx context.Context, token RefreshToken) (Session, error)
	RefreshToken(ctx context.Context)
	IsRevoked(ctx context.Context, hash token.Hash) bool
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
	userSessions map[uuid.UUID][]uuid.UUID

	// hasWorker flags whether this backend has a cleaner worker started
	hasWorker bool

	workerInterval time.Duration
	sync.RWMutex
}

// NewDefaultRegistryBackend initializes a default in-memory registry
func NewDefaultRegistryBackend() *DefaultBackend {
	b := &DefaultBackend{
		sessions:       make(map[uuid.UUID]Session),
		refreshTokens:  make(map[Hash]RefreshToken),
		userSessions:   make(map[uuid.UUID][]uuid.UUID),
		workerInterval: 1 * time.Minute,
	}

	// starting the maintenance worker
	if err := b.startWorker(); err != nil {
		panic(errors.Wrap(err, "AuthRegistryBackend: failed to start worker"))
	}

	return b
}

func (b *DefaultBackend) startWorker() error {
	if b.hasWorker {
		return errors.New("worker has already been started")
	}

	// capturing this instance by a closure
	go func() {
		log.Println("AuthRegistryBackend: worker started")

		b.hasWorker = true
		for {
			// running a blacklist cleanup
			if err := b.cleanup(); err != nil {
				log.Printf("WARNING: auth registry worker has failed to cleanup: %s", err)
			}

			time.Sleep(b.workerInterval)
		}
	}()

	return nil
}

// cleanup performs registry in-memory cleanup over time
// NOTE: this is the default, but not the most optimal approach
func (b *DefaultBackend) cleanup() (err error) {
	b.Lock()
	defer b.Unlock()

	// clearing out expired items
	for jti, expireAt := range b.blacklist {
		if expireAt > timestamp.Now() {
			if err = b.DeleteRevokedAccessItem(jti); err != nil {
				return err
			}
		}
	}

	// clearing out expired userSessions
	for userID, sessionMap := range b.userSessions {
		for tokenHash, s := range sessionMap {
			if s.ExpireAt < timestamp.Now() {
				delete(b.userSessions[userID], tokenHash)
			}
		}
	}

	return nil
}

// PutRevokedAccessToken stores a registry item
func (b *DefaultBackend) PutRevokedAccessToken(item RevokedAccessToken) error {
	if err := item.Validate(); err != nil {
		return err
	}

	b.Lock()
	b.blacklist[item.AccessTokenID] = item
	b.Unlock()

	return nil
}

// IsRevoked returns true if an item with such ObjectID is found
// NOTE: function doesn't care whether this item has expired or not
func (b *DefaultBackend) IsRevoked(tokenID uuid.UUID) bool {
	b.RLock()
	_, ok := b.blacklist[tokenID]
	b.RUnlock()

	return ok
}

// DeleteRevokedAccessItem deletes a revoked item from the registry
func (b *DefaultBackend) DeleteRevokedAccessItem(jti uuid.UUID) error {
	b.Lock()
	delete(b.blacklist, jti)
	b.Unlock()

	return nil
}

// PutSession stores a given session to a temporary registry backend
func (b *DefaultBackend) PutSession(s Session) error {
	if err := s.Validate(); err != nil {
		return err
	}

	b.Lock()

	// initializing a nested map if it hasn't been yet
	if b.userSessions[s.UserID] == nil {
		b.userSessions[s.UserID] = make(map[token.Hash]Session)
	}

	// storing session until it's expired and removed
	b.userSessions[s.UserID][s.Token] = s

	// mapping token to this session
	b.tokenMap[s.Token] = s
	b.jtiMap[s.JTI] = s.JTI

	b.Unlock()

	return nil
}

// GetSession fetches a session by a given token hash from the registry backend
func (b *DefaultBackend) GetSession(token token.Hash) (sess Session, err error) {
	b.RLock()
	sess, ok := b.tokenMap[token]
	b.RUnlock()

	if !ok {
		return sess, ErrSessionNotFound
	}

	return sess, nil
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteSession(sess Session) error {
	if err := sess.Validate(); err != nil {
		return err
	}

	b.Lock()
	delete(b.userSessions[sess.UserID], sess.Token)
	delete(b.tokenMap, sess.Token)
	b.Unlock()

	return nil
}

// GetSessionByAccessToken retrieves session by an accesspolicy token ObjectID (JTI: JWT Hash ObjectID)
func (b *DefaultBackend) GetSessionByAccessToken(jti uuid.UUID) (sess Session, err error) {
	b.RLock()
	sess, ok := b.jtiMap[jti]
	b.RUnlock()

	if !ok {
		return sess, ErrSessionNotFound
	}

	return sess, nil
}

// GetSessionByRefreshToken retrieves session by a refresh token
func (b *DefaultBackend) GetSessionByRefreshToken(rtok string) (sess Session, err error) {
	b.RLock()
	sess, ok := b.refreshTokens[rtok]
	b.RUnlock()

	if !ok {
		return sess, ErrSessionNotFound
	}

	return sess, nil
}
