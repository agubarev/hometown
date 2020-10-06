package auth

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agubarev/hometown/pkg/client"
	"github.com/google/uuid"
)

// TODO: consider storing some client fingerprint inside the session

const (
	SRevokedByLogout uint32 = 1 << iota
	SRevokedByExpiry
	SRevokedBySystem
	SRevokedByClient
	SCreatedByCreds
	SCreatedByRefToken
)

const (
	// DefaultIdleTimeoutSec is the time since the last session activity (in seconds)
	DefaultIdleTimeoutSec = 120

	// DefaultSessionTTL is the default session lifetime
	DefaultSessionTTL = 10 * time.Minute
)

// IdentityKind represents an identity kind (i.e.: user/device/etc...)
type IdentityKind uint8

const (
	IKNone = iota
	IKUser
)

func (k IdentityKind) String() string {
	switch k {
	case IKUser:
		return "user"
	default:
		return "unrecognized identity"
	}
}

// Identity represents a session owner
type Identity struct {
	ID   uuid.UUID    `json:"id"`
	Kind IdentityKind `json:"kind"`
}

// NilIdentity represents a "no identity"
var NilIdentity = Identity{
	ID:   uuid.UUID{},
	Kind: 0,
}

func UserIdentity(id uuid.UUID) Identity { return Identity{ID: id, Kind: IKUser} }

func (ident Identity) Validate() error {
	if ident.Kind == IKNone && ident.ID != uuid.Nil {
		return errors.New("identity id without kind")
	}

	if ident.Kind != IKNone && ident.ID == uuid.Nil {
		return ErrInvalidIdentityID
	}

	return nil
}

// Session represents an authenticated session
type Session struct {
	// ID is also used as JTI (JWT ID)
	ID uuid.UUID `db:"id" json:"id"`

	// ClientID is an ID of a client that originally initiated this session
	ClientID uuid.UUID `db:"client_id" json:"client_id"`

	// Identity is the owner of this session
	Identity Identity `db:"identity" json:"identity"`

	// IP is the IP address from which this session has been initiated
	IP net.IP `db:"ip" json:"ip"`

	// Flags describes metadata like whether it's idling, revoked,
	// revoked by its owner, expiry, client or some external system
	Flags uint32 `db:"flags" json:"flags"`

	// times
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	RefreshedAt time.Time `db:"refreshed_at" json:"refreshed_at"`
	RevokedAt   time.Time `db:"revoked_at" json:"revoked_at"`
	ExpireAt    time.Time `db:"expire_at" json:"expire_at"`

	// this is a unix timestamp (in seconds) which marks
	// the last session activity
	lastActiveAt int64

	sync.RWMutex
}

func NewSession(c *client.Client, ident Identity, meta *RequestMetadata, ttl time.Duration) (s *Session, err error) {
	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	s = &Session{
		ID:        uuid.New(),
		ClientID:  c.ID,
		Identity:  ident,
		IP:        meta.IP,
		Flags:     0,
		CreatedAt: time.Now(),
		ExpireAt:  time.Now().Add(ttl),
	}

	return s, nil
}

func (s *Session) revoke(rflags uint32) error {
	s.Lock()
	defer s.Unlock()

	if !s.RevokedAt.IsZero() {
		return ErrSessionAlreadyRevoked
	}

	if rflags&(SRevokedByClient|SRevokedByLogout|SRevokedBySystem|SRevokedByExpiry) == 0 {
		return ErrInvalidRevocationFlag
	}

	// setting revocation timestamp
	s.RevokedAt = time.Now()

	// updating revocation flags
	for {
		if atomic.CompareAndSwapUint32(&s.Flags, s.Flags, s.Flags|rflags) {
			return nil
		}
	}
}

// SanitizeAndValidate validates the session
func (s *Session) Validate() error {
	if s.ID == uuid.Nil {
		return ErrInvalidSessionID
	}

	if s.ExpireAt.IsZero() {
		return ErrZeroExpiration
	}

	if s.Identity.ID == uuid.Nil {
		return ErrInvalidIdentityID
	}

	return nil
}

// TimeLeft returns the time remaining before it expires
// NOTE: in nanoseconds
func (s *Session) TimeLeft() time.Duration {
	ttl := time.Now().Sub(s.ExpireAt)
	if ttl <= 0 {
		return 0
	}

	return ttl
}

func (s *Session) RevokeByClient() error { return s.revoke(SRevokedByClient) }
func (s *Session) RevokeBySystem() error { return s.revoke(SRevokedBySystem) }
func (s *Session) RevokeByExpiry() error { return s.revoke(SRevokedByExpiry) }
func (s *Session) Touch()                { atomic.StoreInt64(&s.lastActiveAt, time.Now().Unix()) }
func (s *Session) IsIdle() bool          { return atomic.LoadInt64(&s.lastActiveAt) >= DefaultIdleTimeoutSec }

// LastActiveAt returns the time when this client was active last time
func (s *Session) LastActiveAt() time.Time {
	return time.Now().Add(time.Duration(atomic.LoadInt64(&s.lastActiveAt)) * time.Second)
}

func (s *Session) IsValid() bool {
	s.RLock()
	defer s.RUnlock()
	return s.RevokedAt.IsZero() && s.ExpireAt.Before(time.Now())
}

func (s *Session) IsRevoked() bool {
	s.RLock()
	defer s.RUnlock()
	return !s.RevokedAt.IsZero()
}

func (s *Session) IsExpired() bool {
	s.RLock()
	defer s.RUnlock()
	return s.ExpireAt.After(time.Now())
}

func (s *Session) IsRevokedByLogout() bool {
	return atomic.LoadUint32(&s.Flags)&SRevokedByLogout == SRevokedByLogout
}

func (s *Session) IsRevokedByExpiry() bool {
	return atomic.LoadUint32(&s.Flags)&SRevokedByExpiry == SRevokedByExpiry
}

func (s *Session) IsRevokedByClient() bool {
	return atomic.LoadUint32(&s.Flags)&SRevokedByClient == SRevokedByClient
}

func (s *Session) IsRevokedBySystem() bool {
	return atomic.LoadUint32(&s.Flags)&SRevokedBySystem == SRevokedBySystem
}

func (s *Session) IsCreatedByCredentials() bool {
	return atomic.LoadUint32(&s.Flags)&SCreatedByCreds == SCreatedByCreds
}

func (s *Session) IsCreatedByRefToken() bool {
	return atomic.LoadUint32(&s.Flags)&SCreatedByRefToken == SCreatedByRefToken
}
