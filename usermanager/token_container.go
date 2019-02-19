package usermanager

import (
	"encoding/base64"
	"fmt"
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

// NewToken creates a new CSPRNG token
// NOTE: payload is whatever token metadata that can be JSON-encoded for further
// processing by checkin callbacks
func NewToken(k TokenKind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
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
	tokens map[string]*Token
	store  TokenStore
}

// NewTokenContainer returns an initialized token container
func NewTokenContainer(s TokenStore) (*TokenContainer, error) {
	if s == nil {
		return nil, ErrNilTokenStore
	}

	c := &TokenContainer{
		tokens: make(map[string]*Token),
		store:  s,
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
	if _, ok := c.tokens[t.Token]; ok {
		return nil, fmt.Errorf("failed to create new token(%s): duplicate value found %s", k, t.Token)
	}

	// storing new token
	err = c.store.Put(t)
	if err != nil {
		return nil, fmt.Errorf("failed to store new token: %s", err)
	}

	// adding new token to the container
	c.tokens[t.Token] = t

	return t, nil
}

// Get obtains a token from the container or returns ErrTokenNotFound
func (c *TokenContainer) Get(token string) (*Token, error) {
	return c.store.Get(token)
}

// Delete deletes a token from the container
func (c *TokenContainer) Delete(token string) error {
	return c.store.Delete(token)
}

// Checkin check in token for processing
func (c *TokenContainer) Checkin(token string) error {
	panic("not implemented")

	return nil
}

// Cleanup performs a cleanup of container by removing expired tokens
// NOTE: may perform regular full cleanups by iterating over all registered tokens
// or irregular, partial cleanup by checking arbitrary tokens and removing expired ones
func (c *TokenContainer) Cleanup() {
	panic("not implemented")
}

// func (c *TokenContainer) RegisterCheckinCallback
