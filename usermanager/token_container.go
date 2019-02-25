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
type TokenKind uint16

func (k TokenKind) String() string {
	switch k {
	case TkUserEmailConfirmation:
		return "user email confirmation token"
	case TkUserPhoneConfirmation:
		return "user phone confirmation token"
	default:
		return fmt.Sprintf("unrecognized token kind: %d", k)
	}
}

// predefined token kinds
const (
	TkUserEmailConfirmation TokenKind = 1 << iota
	TkUserPhoneConfirmation

	TkAllTokens TokenKind = ^TokenKind(0)
)

// Token represents a general-purpose token
// NOTE: the token will expire after certain conditions are met
// i.e. after specific time or a set number of checkins
// TODO: add complexity variations for different use cases, i.e. SMS code should be short
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
	CheckinRemainder int `json:"c"`
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
	case t.CheckinRemainder == -1: // unlimited checkins
		return nil
	case t.CheckinRemainder > 0: // still has checkins left
		return nil
	case t.CheckinRemainder == 0: // no checkins remaining
		return ErrTokenUsedUp
	}

	return nil
}

// NewToken creates a new CSPRNG token
// NOTE: payload is whatever token metadata that can be JSON-encoded for further
// processing by checkin callbacks
// NOTE: checkin remainder must be -1 (indefinite) or greater than 0 (default: 1)
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
		Kind:             k,
		Token:            base64.URLEncoding.EncodeToString(tokenBuf),
		ExpireAt:         time.Now().Add(ttl),
		CheckinRemainder: checkins,
		Payload:          payloadBuf,
	}

	return t, nil
}

// TokenContainer is a general-purpose token container
type TokenContainer struct {
	// base context for the token callback call chain
	BaseContext context.Context

	tokens    map[string]*Token
	store     TokenStore
	callbacks []TokenCallback
	errorChan chan TokenCallbackError
	sync.RWMutex
}

// TokenCallback is a function metadata
type TokenCallback struct {
	ID       string
	Kind     TokenKind
	Function func(ctx context.Context, t *Token) error
}

// TokenCallbackError represents an error which could be produced by callback
type TokenCallbackError struct {
	Kind  TokenKind
	Token *Token
	Err   error
}

// NewTokenContainer returns an initialized token container
func NewTokenContainer(s TokenStore) (*TokenContainer, error) {
	if s == nil {
		return nil, ErrNilTokenStore
	}

	c := &TokenContainer{
		BaseContext: context.Background(),
		tokens:      make(map[string]*Token),
		store:       s,
		callbacks:   make([]TokenCallback, 0),
		errorChan:   make(chan TokenCallbackError, 100),
	}

	return c, nil
}

// Validate validates token container
func (c *TokenContainer) Validate() error {
	if c == nil {
		return ErrNilTokenContainer
	}

	if c.tokens == nil {
		return fmt.Errorf("token map is not initialized")
	}

	if c.store == nil {
		return ErrNilTokenStore
	}

	if c.callbacks == nil {
		return fmt.Errorf("callback slice is not initialized")
	}

	if c.errorChan == nil {
		return fmt.Errorf("error channel is not initialized")
	}

	return nil
}

// List returns a slice of tokens filtered by a given kind mask
func (c *TokenContainer) List(k TokenKind) []*Token {
	ts := make([]*Token, 0)

	for _, t := range c.tokens {
		if t.Kind&k != 0 {
			ts = append(ts, t)
		}
	}

	return ts
}

// Create initializes, registers and returns a new token
func (c *TokenContainer) Create(k TokenKind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
	t, err := NewToken(k, payload, ttl, checkins)
	if err != nil {
		return nil, fmt.Errorf("failed to create new token(%s): %s", k, err)
	}

	// paranoid check; making sure there is no existing token object with such token key string
	if _, err := c.Get(t.Token); err == nil {
		return nil, fmt.Errorf("failed to create new token(%s): found duplicate token %s", k, t.Token)
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
func (c *TokenContainer) Delete(t *Token) error {
	// clearing token from the map
	c.Lock()
	delete(c.tokens, t.Token)
	c.Unlock()

	// removing from the store
	if err := c.store.Delete(t.Token); err != nil {
		return err
	}

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

	// base context for this call chain
	ctx := context.Background()

	// obtaining callbacks for this kind
	callbacks := c.GetCallbacks(t.Kind)
	if len(callbacks) == 0 {
		// there's no point proceeding without callbacks
		return ErrTokenCallbackNotFound
	}

	// obtaining a list of callbacks attached to this token kind
	// NOTE: this process is synchronous because the caller expects a response
	// NOTE: checkin stops at first error returned by any callback because they're set for a reason
	for _, cb := range callbacks {
		if err = cb.Function(ctx, t); err != nil {
			return fmt.Errorf("checkin failed (token=%s, kind=%s): %s", t.Token, t.Kind, err)
		}
	}

	// post-checkin operations
	if t.CheckinRemainder > 0 {
		t.CheckinRemainder--
	}

	// now if validate returns an error, this means that this token is void,
	// and can be safely removed, because all related callbacks ran successfully at this point
	if err = t.Validate(); err != nil {
		if err := c.Delete(t); err != nil {
			return fmt.Errorf("failed to delete token(%s) after use: %s", t.Token, err)
		}
	}

	return nil
}

// Cleanup performs a full cleanup of the container by removing tokens
// which failed to pass validation
func (c *TokenContainer) Cleanup() error {
	c.Lock()
	for _, t := range c.tokens {
		if err := t.Validate(); err != nil {
			// ignoring errors from delete, because some tokens could possibly be
			// already deleted other way i.e. records expired by the store
			c.Delete(t)
		}
	}
	c.Unlock()

	return nil
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
		if cb.ID == id {
			return ErrTokenDuplicateCallbackID
		}
	}

	c.callbacks = append(c.callbacks, TokenCallback{
		ID:       id,
		Kind:     k,
		Function: fn,
	})

	return nil
}

// GetCallback returns a named callback if it exists
func (c *TokenContainer) GetCallback(id string) (*TokenCallback, error) {
	// just looping over the slice
	for _, cb := range c.callbacks {
		if cb.ID == strings.ToLower(id) {
			// returning pointer to callback object
			return &cb, nil
		}
	}

	return nil, ErrTokenCallbackNotFound
}

// GetCallbacks returns callback stack by a given token kind
func (c *TokenContainer) GetCallbacks(k TokenKind) []TokenCallback {
	cbstack := make([]TokenCallback, 0)
	for _, cb := range c.callbacks {
		if cb.Kind&k != 0 {
			cbstack = append(cbstack, cb)
		}
	}

	return cbstack
}

// RemoveCallback removes token callback by ID, returns ErrTokenCallbackNotfound
func (c *TokenContainer) RemoveCallback(id string) error {
	id = strings.ToLower(id)

	c.Lock()
	defer c.Unlock()

	// finding and removing callback
	for i, cb := range c.callbacks {
		if cb.ID == id {
			c.callbacks = append(c.callbacks[0:i], c.callbacks[i+1:]...)
			return nil
		}
	}

	return ErrTokenCallbackNotFound
}
