package token

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/agubarev/hometown/pkg/util"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// errors
var (
	ErrNilDatabase              = errors.New("database is nil")
	ErrNilTokenStore            = errors.New("token store is nil")
	ErrNilTokenOwner            = errors.New("token owner is nil")
	ErrNilToken                 = errors.New("token is nil")
	ErrTokenNotFound            = errors.New("token not found")
	ErrTokenExpired             = errors.New("token is expired")
	ErrTokenUsedUp              = errors.New("token is all used up")
	ErrTokenDuplicateCallbackID = errors.New("token callback id is already registered")
	ErrTokenCallbackNotFound    = errors.New("token callback not found")
	ErrNilTokenManager          = errors.New("token manager is nil")
	ErrNothingChanged           = errors.New("nothing changed")
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Length total length in bytes (including prefix, id and random bytes)
const Length = 30

// DefaultTTL defines the default token longevity duration from the moment of its creation
const DefaultTTL = 1 * time.Hour

// GroupMemberKind represents the type of a token, used by a token container
type Kind uint16

func (k Kind) String() string {
	switch k {
	case TkRefreshToken:
		return "refresh token"
	case TkUserEmailConfirmation:
		return "email confirmation token"
	case TkUserPhoneConfirmation:
		return "phone confirmation token"
	default:
		return fmt.Sprintf("unrecognized token kind: %d", k)
	}
}

// predefined token kinds
const (
	TkRefreshToken Kind = 1 << iota
	TkSessionToken
	TkUserEmailConfirmation
	TkUserPhoneConfirmation

	TkAllTokens Kind = ^Kind(0)
)

// Token represents a general-purpose token
// NOTE: the token will expire after certain conditions are met
// i.e. after specific time or a set number of checkins
// TODO add complexity variations for different use cases, i.e. SMS code should be short
type Token struct {
	// the kind of operation this token is associated to
	Kind Kind `db:"kind" json:"kind"`

	// token string id
	Token string `db:"token" json:"token"`

	// the accompanying metadata
	Payload []byte `db:"payload" json:"payload"`

	// holds the initial checkin threshold number
	CheckinTotal int `db:"checkin_total" json:"checkin_total"`

	// holds how many checkins could be performed before it's void
	CheckinRemainder int `db:"checkin_remainder" json:"checkin_remainder"`

	// time when this token was created
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// denotes when this token becomes void and is removed
	ExpireAt time.Time `db:"expire_at" json:"expire_at"`
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
func NewToken(k Kind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
	if checkins == 0 {
		return nil, fmt.Errorf("failed to initialize new token: checkins remainder must be -1 or greater than 0")
	}

	// setting given ttl (time to live) if given duration is greater than zero,
	// otherwise using default token longevity
	// NOTE: the final expiration time is the current time plus ttl duration
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	// marshaling payload
	payloadBuf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// generating token string
	tokenBuf, err := util.NewCSPRNG(Length)
	if err != nil {
		return nil, err
	}

	t := &Token{
		Kind:             k,
		Token:            base64.URLEncoding.EncodeToString(tokenBuf),
		CheckinTotal:     checkins,
		CheckinRemainder: checkins,
		Payload:          payloadBuf,
		CreatedAt:        time.Now(),
		ExpireAt:         time.Now().Add(ttl),
	}

	return t, nil
}

// userManager is a general-purpose token container
type Manager struct {
	// base context for the token callback call chain
	BaseContext context.Context

	tokens    map[string]*Token
	store     Store
	callbacks []Callback
	errorChan chan CallbackError
	logger    *zap.Logger
	sync.RWMutex
}

// Callback is a function metadata
type Callback struct {
	ID       string
	Kind     Kind
	Function func(ctx context.Context, t *Token) error
}

// CallbackError represents an error which could be produced by callback
type CallbackError struct {
	Kind  Kind
	Token *Token
	Err   error
}

// NewTokenManager returns an initialized token container
func NewTokenManager(s Store) (*Manager, error) {
	if s == nil {
		return nil, ErrNilTokenStore
	}

	c := &Manager{
		BaseContext: context.Background(),
		tokens:      make(map[string]*Token),
		store:       s,
		callbacks:   make([]Callback, 0),
		errorChan:   make(chan CallbackError, 100),
	}

	return c, nil
}

// SetLogger assigns a logger to this manager
func (m *Manager) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named("[token]")
	}

	m.logger = logger

	return nil
}

// Logger returns own logger
func (m *Manager) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to initialize token manager logger: %s", err))
		}

		m.logger = l
	}

	return m.logger
}

// Init initializes group manager
func (m *Manager) Init() error {
	if err := m.Validate(); err != nil {
		return err
	}

	// not doing anything at the moment

	return nil
}

// Store returns store if set
func (m *Manager) Store() (Store, error) {
	if m.store == nil {
		return nil, ErrNilTokenStore
	}

	return m.store, nil
}

// Validate validates token container
func (m *Manager) Validate() error {
	if m == nil {
		return ErrNilTokenManager
	}

	if m.tokens == nil {
		return fmt.Errorf("token map is not initialized")
	}

	if m.store == nil {
		return ErrNilTokenStore
	}

	if m.callbacks == nil {
		return fmt.Errorf("callback slice is not initialized")
	}

	if m.errorChan == nil {
		return fmt.Errorf("error channel is not initialized")
	}

	return nil
}

// List returns a slice of tokens filtered by a given kind mask
func (m *Manager) List(k Kind) []*Token {
	ts := make([]*Token, 0)

	for _, t := range m.tokens {
		if t.Kind&k != 0 {
			ts = append(ts, t)
		}
	}

	return ts
}

// Upsert initializes, registers and returns a new token
func (m *Manager) Create(ctx context.Context, k Kind, payload interface{}, ttl time.Duration, checkins int) (*Token, error) {
	t, err := NewToken(k, payload, ttl, checkins)
	if err != nil {
		return nil, fmt.Errorf("failed to create new token(%s): %s", k, err)
	}

	// paranoid check; making sure there is no existing token object with such token key string
	if _, err := m.Get(ctx, t.Token); err == nil {
		return nil, fmt.Errorf("failed to create new token(%s): found duplicate token %s", k, t.Token)
	}

	// storing new token
	err = m.store.Put(ctx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to store new token: %s", err)
	}

	// adding new token to the container
	m.Lock()
	m.tokens[t.Token] = t
	m.Unlock()

	return t, nil
}

// Get obtains a token from the container or returns ErrTokenNotFound
func (m *Manager) Get(ctx context.Context, token string) (*Token, error) {
	// checking map cache first
	m.RLock()
	t, ok := m.tokens[token]
	m.RUnlock()

	// found cached token
	if ok {
		return t, nil
	}

	// checking the store
	t, err := m.store.Get(ctx, token)
	if err != nil {
		return nil, err
	}

	// adding token to the map and returning
	m.Lock()
	m.tokens[t.Token] = t
	m.Unlock()

	return t, nil
}

// Delete deletes a token from the container
func (m *Manager) Delete(ctx context.Context, t *Token) error {
	// clearing token from the map
	m.Lock()
	delete(m.tokens, t.Token)
	m.Unlock()

	// removing from the store
	if err := m.store.DeleteByToken(ctx, t.Token); err != nil {
		return err
	}

	return nil
}

// Checkin check in token for processing
func (m *Manager) Checkin(ctx context.Context, token string) error {
	t, err := m.Get(context.Background(), token)
	if err != nil {
		return err
	}

	if err = t.Validate(); err != nil {
		return err
	}

	// obtaining callbacks for this kind
	callbacks := m.GetCallbacks(t.Kind)
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
		if err := m.Delete(context.Background(), t); err != nil {
			return fmt.Errorf("failed to delete token(%s) after use: %s", t.Token, err)
		}
	}

	return nil
}

// Cleanup performs a full cleanup of the container by removing tokens
// which failed to pass validation
func (m *Manager) Cleanup(ctx context.Context) (err error) {
	m.Lock()
	for _, t := range m.tokens {
		if err := t.Validate(); err != nil {
			// ignoring errors from delete, because some tokens could possibly be
			// already deleted other way i.e. records expired by the store
			if err = m.Delete(ctx, t); err != nil {
				m.Logger().Warn("failed to delete token", zap.String("token", t.Token), zap.Error(err))
			}
		}
	}
	m.Unlock()

	return nil
}

// AddCallback adds callback function to container's callstack to be called upon token checkins
func (m *Manager) AddCallback(k Kind, id string, fn func(ctx context.Context, t *Token) error) error {
	// this is straightforward, adding function to a callback stack
	// NOTE: GroupMemberID is a basic mechanism to prevent multiple callback additions
	id = strings.ToLower(id)

	m.Lock()
	defer m.Unlock()

	// making sure there isn't any callback registered with this GroupMemberID
	for _, cb := range m.callbacks {
		if cb.ID == id {
			return ErrTokenDuplicateCallbackID
		}
	}

	m.callbacks = append(m.callbacks, Callback{
		ID:       id,
		Kind:     k,
		Function: fn,
	})

	return nil
}

// GetCallback returns a named callback if it exists
func (m *Manager) GetCallback(id string) (*Callback, error) {
	// just looping over the slice
	for _, cb := range m.callbacks {
		if cb.ID == strings.ToLower(id) {
			// returning pointer to callback object
			return &cb, nil
		}
	}

	return nil, ErrTokenCallbackNotFound
}

// GetCallbacks returns callback stack by a given token kind
func (m *Manager) GetCallbacks(k Kind) []Callback {
	cbstack := make([]Callback, 0)
	for _, cb := range m.callbacks {
		if cb.Kind&k != 0 {
			cbstack = append(cbstack, cb)
		}
	}

	return cbstack
}

// RemoveCallback removes token callback by GroupMemberID, returns ErrTokenCallbackNotfound
func (m *Manager) RemoveCallback(id string) error {
	id = strings.ToLower(id)

	m.Lock()
	defer m.Unlock()

	// finding and removing callback
	for i, cb := range m.callbacks {
		if cb.ID == id {
			m.callbacks = append(m.callbacks[0:i], m.callbacks[i+1:]...)
			return nil
		}
	}

	return ErrTokenCallbackNotFound
}
