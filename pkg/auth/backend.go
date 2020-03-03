package auth

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// Backend is an interface contract for an auth backend
type Backend interface {
	PutRevokedAccessToken(RevokedAccessToken) error
	IsRevoked(string) bool
	DeleteRevokedAccessItem(string) error
	PutSession(Session) error
	GetSession(string) (Session, error)
	GetSessionByAccessToken(string) (Session, error)
	GetSessionByRefreshToken(string) (Session, error)
	DeleteSession(Session) error
}

// DefaultBackend is a default in-memory implementation
type DefaultBackend struct {
	// blacklist is a map of revoked access token IDs
	blacklist map[string]RevokedAccessToken

	// session token map, token to session
	stokenMap map[string]Session

	// access token map, token ObjectID (jti) to session
	jtiMap map[string]Session

	// refresh token map, token to session
	rtokenMap map[string]Session

	// is a map of user IDs to a map of tokens, containing the actual session
	sessions map[int64]map[string]Session

	// hasWorker flags whether this backend has a cleaner worker started
	hasWorker bool

	workerInterval time.Duration
	sync.RWMutex
}

// NewDefaultRegistryBackend initializes a default in-memory registry
func NewDefaultRegistryBackend() *DefaultBackend {
	b := &DefaultBackend{
		blacklist:      make(map[string]RevokedAccessToken),
		stokenMap:      make(map[string]Session),
		sessions:       make(map[int64]map[string]Session),
		workerInterval: 1 * time.Minute,
	}

	// starting the maintenance worker
	if err := b.startWorker(); err != nil {
		panic(fmt.Errorf("AuthRegistryBackend: failed to start worker: %s", err))
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
func (b *DefaultBackend) cleanup() error {
	b.Lock()
	defer b.Unlock()

	// clearing out expired items
	for k, v := range b.blacklist {
		if v.ExpireAt.After(time.Now()) {
			if err := b.DeleteRevokedAccessItem(k); err != nil {
				return err
			}
		}
	}

	// clearing out expired sessions
	for uid, smap := range b.sessions {
		for stok, s := range smap {
			if s.ExpireAt.Before(time.Now()) {
				delete(b.sessions[uid], stok)
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
	b.blacklist[item.TokenID] = item
	b.Unlock()

	return nil
}

// IsRevoked returns true if an item with such ObjectID is found
// NOTE: function doesn't care whether this item has expired or not
func (b *DefaultBackend) IsRevoked(tokenID string) bool {
	b.RLock()
	_, ok := b.blacklist[tokenID]
	b.RUnlock()

	return ok
}

// DeleteRevokedAccessItem deletes a revoked item from the registry
func (b *DefaultBackend) DeleteRevokedAccessItem(tokenID string) error {
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
	if b.sessions[s.UserID] == nil {
		b.sessions[s.UserID] = make(map[string]Session)
	}

	// storing session until it's expired and removed
	b.sessions[s.UserID][s.Token] = s

	// mapping token to this session
	b.stokenMap[s.Token] = s
	b.stokenMap[s.AccessTokenID] = s

	b.Unlock()

	return nil
}

// GetSession fetches a session by a given token string,
// from the registry backend
func (b *DefaultBackend) GetSession(stok string) (Session, error) {
	b.RLock()
	s, ok := b.stokenMap[stok]
	b.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	return s, nil
}

// DeleteSession deletes a given session from the backend registry
func (b *DefaultBackend) DeleteSession(s Session) error {
	if err := s.Validate(); err != nil {
		return err
	}

	b.Lock()
	delete(b.sessions[s.UserID], s.Token)
	delete(b.stokenMap, s.Token)
	b.Unlock()

	return nil
}

// GetSessionByAccessToken retrieves session by an access token ObjectID (JTI: JWT Token ObjectID)
func (b *DefaultBackend) GetSessionByAccessToken(jti string) (Session, error) {
	b.RLock()
	s, ok := b.jtiMap[jti]
	b.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	return s, nil
}

// GetSessionByRefreshToken retrieves session by a refresh token
func (b *DefaultBackend) GetSessionByRefreshToken(rtok string) (Session, error) {
	b.RLock()
	s, ok := b.rtokenMap[rtok]
	b.RUnlock()

	if !ok {
		return nil, ErrSessionNotFound
	}

	return s, nil
}
