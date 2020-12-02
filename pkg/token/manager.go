package token

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// errors
var (
	ErrNilDatabase              = errors.New("database is nil")
	ErrNilTokenStore            = errors.New("token store is nil")
	ErrEmptyTokenHash           = errors.New("token hash is empty")
	ErrTokenNotFound            = errors.New("token not found")
	ErrTokenExpired             = errors.New("token is expired")
	ErrTokenUsedUp              = errors.New("token is all used up")
	ErrTokenDuplicateCallbackID = errors.New("token callback id is already registered")
	ErrTokenCallbackNotFound    = errors.New("token callback not found")
	ErrNilTokenManager          = errors.New("token manager is nil")
	ErrNothingChanged           = errors.New("nothing changed")
	ErrDuplicateToken           = errors.New("duplicate token")
)

// Length total length in bytes (including prefix, id and random bytes)
const Length = 32

// DefaultTTL defines the default token longevity duration from the moment of its creation
const DefaultTTL = 1 * time.Hour

// Hash represents a token hash
type Hash [Length]byte

func NewHash() (h Hash) {
	// generating token hash
	_, err := rand.Read(h[:])
	if err != nil {
		return h
	}

	return h
}

func (h Hash) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if h[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(h[:], byte(0))
	if zpos == -1 {
		return append(buf, h[:]...), nil
	}

	return append(buf, h[0:zpos]...), nil
}

func (h Hash) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if src == nil {
		return nil
	}

	copy(h[:], src)

	return nil
}

func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

type CallbackName [32]byte

func NewCallbackName(s string) (name CallbackName) {
	copy(name[:], bytes.ToLower(bytes.TrimSpace([]byte(s))))
	return name
}

func (name CallbackName) Scan(data interface{}) error {
	copy(name[:], data.([]byte))
	return nil
}

func (name CallbackName) Value() (driver.Value, error) {
	if name[0] == 0 {
		return "", nil
	}

	// finding position of zero
	zeroPos := bytes.IndexByte(name[:], byte(0))
	if zeroPos == -1 {
		return name[:], nil
	}

	return name[0:zeroPos], nil
}

// ObjectKind represents the type of a token, used by a token container
type Kind uint16

func (k Kind) String() string {
	switch k {
	case TRefreshToken:
		return "refresh token"
	case TEmailConfirmation:
		return "email confirmation token"
	case TPhoneConfirmation:
		return "phone confirmation token"
	default:
		return fmt.Sprintf("unrecognized token kind: %d", k)
	}
}

// predefined token kinds
const (
	TRefreshToken Kind = 1 << iota
	TSessionToken
	TEmailConfirmation
	TPhoneConfirmation

	TAll = ^Kind(0)
)

// Hash represents a general-purpose token
// NOTE: the token will expire after certain conditions are met
// i.e. after specific time or a set number of checkins
// TODO add complexity variations for different use cases, i.e. SMS code should be short
type Token struct {
	// the kind of operation this token is associated to
	Kind Kind `db:"kind" json:"kind"`

	// token string id
	Hash Hash `db:"hash" json:"hash"`

	// holds the initial checkin threshold number
	CheckinTotal int32 `db:"checkin_total" json:"checkin_total"`

	// holds how many checkins could be performed before it's void
	CheckinRemainder int32 `db:"checkin_remainder" json:"checkin_remainder"`

	// time when this token was created
	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// denotes when this token becomes void and is removed
	ExpireAt time.Time `db:"expire_at" json:"expire_at"`
}

// SanitizeAndValidate checks whether the token is expired or ran out of checkins left
// NOTE: returns errors instead of booleans only for more flexible explicitness
func (t Token) Validate() error {
	// checking whether token's expiration time is behind current moment
	if t.ExpireAt.After(time.Now()) {
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

// New creates a new token object with CSPRNG hash
// NOTE: payload is any token metadata that can be JSON-encoded for further
// processing by checkin callbacks
// NOTE: checkin remainder must be -1 (indefinite) or greater than 0 (default: 1)
func New(k Kind, ttl time.Duration, checkins int32) (t Token, err error) {
	if checkins == 0 {
		return t, errors.New("failed to initialize new token: checkins remainder must be -1 or greater than 0")
	}

	// setting given ttl (time to live in seconds) if given duration is greater than zero,
	// otherwise using default token longevity
	// NOTE: the final expiration time is the current time plus ttl duration
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	// current timestamp
	now := time.Now()

	t = Token{
		Kind:             k,
		Hash:             NewHash(),
		CheckinTotal:     checkins,
		CheckinRemainder: checkins,
		CreatedAt:        now,
		ExpireAt:         now.Add(ttl),
	}

	return t, nil
}

// userManager is a general-purpose token container
type Manager struct {
	// base context for the token callback call chain
	BaseContext context.Context

	tokens    map[Hash]Token
	store     Store
	callbacks []Callback
	errorChan chan CallbackError
	logger    *zap.Logger
	sync.RWMutex
}

// Callback is a function metadata
type Callback struct {
	Name     CallbackName
	Kind     Kind
	Function func(ctx context.Context, t Token) error
}

// CallbackError represents an error which could be produced by callback
type CallbackError struct {
	Kind  Kind
	Token Token
	Err   error
}

// NewManager returns an initialized token container
func NewManager(s Store) (*Manager, error) {
	if s == nil {
		return nil, ErrNilTokenStore
	}

	c := &Manager{
		BaseContext: context.Background(),
		tokens:      make(map[Hash]Token),
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

// SanitizeAndValidate validates token container
func (m *Manager) Validate() error {
	if m == nil {
		return ErrNilTokenManager
	}

	if m.tokens == nil {
		return errors.New("token map is not initialized")
	}

	if m.store == nil {
		return ErrNilTokenStore
	}

	if m.callbacks == nil {
		return errors.New("callback slice is not initialized")
	}

	if m.errorChan == nil {
		return errors.New("error channel is not initialized")
	}

	return nil
}

// Registry returns a slice of tokens filtered by a given kind mask
func (m *Manager) List(k Kind) []Token {
	ts := make([]Token, 0)

	for _, t := range m.tokens {
		if t.Kind&k != 0 {
			ts = append(ts, t)
		}
	}

	return ts
}

// Upsert initializes, registers and returns a new token
func (m *Manager) Create(ctx context.Context, k Kind, ttl time.Duration, checkins int32) (t Token, err error) {
	t, err = New(k, ttl, checkins)
	if err != nil {
		return t, errors.Wrapf(err, "failed to initialize new token: %s", k)
	}

	// paranoid check; making sure there is no existing token object with such token key string
	if _, err = m.Get(ctx, t.Hash); err == nil {
		return t, ErrDuplicateToken
	}

	// storing new token
	err = m.store.Put(ctx, t)
	if err != nil {
		return t, errors.Wrap(err, "failed to store new token")
	}

	// adding new token to the container
	m.Lock()
	m.tokens[t.Hash] = t
	m.Unlock()

	return t, nil
}

// Get obtains a token from the container or returns ErrTokenNotFound
func (m *Manager) Get(ctx context.Context, hash Hash) (t Token, err error) {
	// checking map cache first
	m.RLock()
	t, ok := m.tokens[hash]
	m.RUnlock()

	// found cached hash
	if ok {
		return t, nil
	}

	// checking the store
	t, err = m.store.Get(ctx, hash)
	if err != nil {
		return t, err
	}

	// adding hash to the map and returning
	m.Lock()
	m.tokens[t.Hash] = t
	m.Unlock()

	return t, nil
}

// DeletePolicy deletes a token from the container
func (m *Manager) Delete(ctx context.Context, t Token) error {
	// clearing token from the map
	m.Lock()
	delete(m.tokens, t.Hash)
	m.Unlock()

	// removing from the store
	if err := m.store.Delete(ctx, t.Hash); err != nil {
		return err
	}

	return nil
}

// Checkin check in token for processing
func (m *Manager) Checkin(ctx context.Context, hash Hash) error {
	t, err := m.Get(context.Background(), hash)
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

	// obtaining a list of callbacks attached to this hash kind
	// NOTE: this process is synchronous because the caller expects a response
	// NOTE: checkin stops at first error returned by any callback because they're set for a reason
	for _, cb := range callbacks {
		if err = cb.Function(ctx, t); err != nil {
			return fmt.Errorf("checkin failed (hash=%s, kind=%s): %s", t.Hash, t.Kind, err)
		}
	}

	// post-checkin operations
	if t.CheckinRemainder > 0 {
		t.CheckinRemainder--
	}

	// now if validate returns an error, this means that this hash is void,
	// and can be safely removed, because all related callbacks ran successfully at this point
	if err = t.Validate(); err != nil {
		if err := m.Delete(context.Background(), t); err != nil {
			return errors.Wrapf(err, "failed to delete depleted token: %s", t.Hash.String())
		}

		return nil
	}

	// TODO: persist changes to the store

	// otherwise, replacing token cache
	m.Lock()
	m.tokens[t.Hash] = t
	m.Unlock()

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
				m.Logger().Warn("failed to delete token", zap.String("token", t.Hash.String()), zap.Error(err))
			}
		}
	}
	m.Unlock()

	return nil
}

// AddCallback adds callback function to container's callstack to be called upon token checkins
func (m *Manager) AddCallback(kind Kind, name CallbackName, fn func(ctx context.Context, t Token) error) error {
	// trimming and flattening case
	copy(name[:], bytes.ToLower(bytes.TrimSpace(name[:])))

	m.Lock()
	defer m.Unlock()

	// making sure there isn't any other callback registered with this name
	for _, cb := range m.callbacks {
		if cb.Name == name {
			return ErrTokenDuplicateCallbackID
		}
	}

	// this is straightforward, just adding function to the callback stack
	m.callbacks = append(m.callbacks, Callback{
		Name:     name,
		Kind:     kind,
		Function: fn,
	})

	return nil
}

// GetCallback returns a named callback if it exists
func (m *Manager) GetCallback(name CallbackName) (*Callback, error) {
	// just looping over the slice
	for _, cb := range m.callbacks {
		if cb.Name == name {
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

// RemoveCallback removes token callback by ObjectID, returns ErrTokenCallbackNotfound
func (m *Manager) RemoveCallback(name CallbackName) error {
	m.Lock()
	defer m.Unlock()

	// finding and removing callback
	for i, cb := range m.callbacks {
		if cb.Name == name {
			m.callbacks = append(m.callbacks[0:i], m.callbacks[i+1:]...)
			return nil
		}
	}

	return ErrTokenCallbackNotFound
}
