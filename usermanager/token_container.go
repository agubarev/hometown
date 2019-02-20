package usermanager

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"gitlab.com/agubarev/hometown/util"
)

// TokenLength total length in bytes (including prefix, id and random bytes)
const TokenLength = 30

// TokenDefaultTTL defines the default token longevity duration from the moment of its creation
const TokenDefaultTTL = 1 * time.Hour

// TokenKind represents the type of a token, used by a token container
type TokenKind uint8

func (k TokenKind) String() string {
	switch k {
	case TkUserConfirmation:
		return "user confirmation token"
	default:
		return fmt.Sprintf("unrecognized token kind: %d", k)
	}
}

// predefined token kinds
const (
	TkUserConfirmation TokenKind = 1 << iota
)

// Token represents a general-purpose token
// NOTE: the token will expire after certain conditions are met
// i.e. after specific time or a set number of checkins
type Token struct {
	// the kind of operation this token is associated to
	Kind TokenKind `json:"k"`

	// token string id
	Token string `json:"t"`

	// the accompanying metadata
	Payload []byte `json:"p"`

	// denotes when this token becomes void and is removed
	ExpireAt time.Time `json:"e"`

	// holds how many checkins could be performed before it's void
	CheckinsLeft int `json:"c"`
}

// Validate checks whether the token is expired or ran out of checkins left
// NOTE: returns errors instead of booleans only for more flexible explicitness
func (t *Token) Validate() error {
	// checking whether token's expiration time is behind current moment
	if t.ExpireAt.Before(time.Now()) {
		return ErrTokenExpired
	}

	// NOTE: now this is important because every token must have an expiration time
	// and the initial number of checkins remaining must always be above 0, unless
	// you want this token to be checkinable indefinitely then the checkins remaining
	// must be -1 and it is very important not to breach this mark of 0.
	// to sum it up: -1 means indefinite, above 0 is how many more times token can be
	// checked in and 0 means void (irrelevant whether it still hasn't expired on a
	// time basis)
	switch true {
	case t.CheckinsLeft == -1: // unlimited checkins
		return nil
	case t.CheckinsLeft > 0: // still has checkins left
		return nil
	case t.CheckinsLeft == 0: // no checkins remaining
		return ErrTokenUsedUp
	}

	return nil
}

// NewToken creates a new CSPRNG token
// NOTE: payload is whatever token metadata that can be JSON-encoded for further
// processing by checkin callbacks
// NOTE: checkin remainder must be -1 (indefinite) or greater than 0
func NewToken(k TokenKind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
	if checkins == 0 {
		return nil, fmt.Errorf("failed to initialize new token: checkins remainder must be -1 or greater than 0")
	}

	// setting given ttl (time to live) if given duration is greater than zero,
	// otherwise using default token longevity
	// NOTE: the final expiration time is the current time plus ttl duration
	if ttl <= 0 {
		ttl = TokenDefaultTTL
	}

	// marshaling payload
	payloadBuf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// generating token string
	tokenBuf, err := util.NewCSPRNG(TokenLength)
	if err != nil {
		return nil, err
	}

	t := &Token{
		Kind:         k,
		Token:        base64.URLEncoding.EncodeToString(tokenBuf),
		ExpireAt:     time.Now().Add(ttl),
		CheckinsLeft: checkins,
		Payload:      payloadBuf,
	}

	return t, nil
}

// TokenContainer is a general-purpose token container
type TokenContainer struct {
	tokens    map[string]*Token
	store     TokenStore
	callbacks []tokenCallback
	errorChan chan tokenError
	sync.RWMutex
}

type tokenCallback struct {
	id       string
	kind     TokenKind
	callback func(ctx context.Context, t *Token) error
}

type tokenError struct {
	kind  TokenKind
	token *Token
	err   error
}

// NewTokenContainer returns an initialized token container
func NewTokenContainer(s TokenStore) (*TokenContainer, error) {
	if s == nil {
		return nil, ErrNilTokenStore
	}

	c := &TokenContainer{
		tokens:    make(map[string]*Token),
		store:     s,
		callbacks: make([]tokenCallback, 0),
		errorChan: make(chan tokenError, 100),
	}

	return c, nil
}

// Create initializes, registers and returns a new token
func (c *TokenContainer) Create(k TokenKind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
	t, err := NewToken(k, payload, ttl, checkins)
	if err != nil {
		return nil, fmt.Errorf("failed to create new token(%s): %s", k, err)
	}

	// checking whether it's already registered
	if _, err := c.Get(t.Token); err == nil {
		return nil, fmt.Errorf("failed to create new token(%s): duplicate value found %s", k, t.Token)
	}

	// storing new token
	err = c.store.Put(t)
	if err != nil {
		return nil, fmt.Errorf("failed to store new token: %s", err)
	}

	// adding new token to the container
	c.Lock()
	c.tokens[t.Token] = t
	c.Unlock()

	return t, nil
}

// Get obtains a token from the container or returns ErrTokenNotFound
func (c *TokenContainer) Get(token string) (*Token, error) {
	// checking map cache first
	c.RLock()
	t, ok := c.tokens[token]
	c.RUnlock()

	// found cached token
	if ok {
		return t, nil
	}

	// checking the store
	t, err := c.store.Get(token)
	if err != nil {
		return nil, err
	}

	// adding token to the map and returning
	c.Lock()
	c.tokens[t.Token] = t
	c.Unlock()

	return t, nil
}

// Delete deletes a token from the container
func (c *TokenContainer) Delete(token string) error {
	if err := c.store.Delete(token); err != nil {
		return err
	}

	// clearing token from the map
	c.Lock()
	delete(c.tokens, token)
	c.Unlock()

	return nil
}

// Checkin check in token for processing
func (c *TokenContainer) Checkin(token string) error {
	t, err := c.Get(token)
	if err != nil {
		return err
	}

	if err = t.Validate(); err != nil {
		return err
	}

	return nil
}

// Cleanup performs a cleanup of container by removing expired tokens
// NOTE: may perform regular full cleanups by iterating over all registered tokens
// or irregular, partial cleanup by checking arbitrary tokens and removing expired ones
func (c *TokenContainer) Cleanup() {
	panic("not implemented")
}

// AddCallback adds callback function to container's callstack to be called upon token checkins
func (c *TokenContainer) AddCallback(k TokenKind, id string, fn func(ctx context.Context, t *Token) error) error {
	// this is straightforward, adding function to a callback stack
	// NOTE: ID is a basic mechanism to prevent multiple callback additions
	id = strings.ToLower(id)

	c.Lock()
	defer c.Unlock()

	// making sure there isn't any callback registered with this ID
	for _, cb := range c.callbacks {
		if cb.id == id {
			return ErrTokenDuplicateCallbackID
		}
	}

	c.callbacks = append(c.callbacks, tokenCallback{
		id:       id,
		kind:     k,
		callback: fn,
	})

	return nil
}

// RemoveCallback removes token callback by ID, returns ErrTokenCallbackNotfound
func (c *TokenContainer) RemoveCallback(id string) error {
	id = strings.ToLower(id)

	// finding and removing callback
	c.Lock()
	for i, cb := range c.callbacks {
		if cb.id == id {
			c.callbacks = append(c.callbacks[0:i], c.callbacks[i+1:]...)
			return nil
		}
	}
	c.Unlock()

	return ErrTokenCallbackNotFound
}
