package auth

import (
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
	PutRevokedAccessToken(RevokedAccessToken) error
	IsRevoked(hash token.Hash) bool
	DeleteRevokedAccessItem(hash token.Hash) error
	PutSession(Session) error
	GetSession(hash token.Hash) (Session, error)
	GetSessionByAccessToken(hash token.Hash) (Session, error)
	GetSessionByRefreshToken(hash token.Hash) (Session, error)
	DeleteSession(Session) error
}

// DefaultBackend is a default in-memory implementation
type DefaultBackend struct {
	// blacklist is a map of revoked accesspolicy token IDs
	blacklist map[uuid.UUID]RevokedAccessToken

	// session token map, token hash to session ID
	tokenMap map[token.Hash]uuid.UUID

	// access token ID (JTI) to session ID
	jtiMap map[uuid.UUID]uuid.UUID

	// refresh token map, token to session ID
	rtokenMap map[token.Hash]uuid.UUID

	// a map of JTI to an actual session
	sessions map[uuid.UUID]Session

	// is a map of user IDs to a map of tokens, containing the actual session
	// NOTE: this is a map of { user ID -> token hash -> session ID (JTI) }
	userSessions map[uuid.UUID]map[token.Hash]uuid.UUID

	// hasWorker flags whether this backend has a cleaner worker started
	hasWorker bool

	workerInterval time.Duration
	sync.RWMutex
}

// NewDefaultRegistryBackend initializes a default in-memory registry
func NewDefaultRegistryBackend() *DefaultBackend {
	b := &DefaultBackend{
		blacklist:      make(map[token.Hash]RevokedAccessToken),
		tokenMap:       make(map[token.Hash]uuid.UUID),
		userSessions:   make(map[uuid.UUID]map[token.Hash]uuid.UUID),
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
	for k, v := range b.blacklist {
		if v.ExpireAt > timestamp.Now() {
			if err = b.DeleteRevokedAccessItem(k); err != nil {
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
func (b *DefaultBackend) DeleteRevokedAccessItem(tokenID uuid.UUID) error {
	b.Lock()
	delete(b.blacklist, tokenID)
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
	b.jtiMap[s.AccessTokenID] = s.AccessTokenID

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
	sess, ok := b.rtokenMap[rtok]
	b.RUnlock()

	if !ok {
		return sess, ErrSessionNotFound
	}

	return sess, nil
}
