package auth

import (
	"sync/atomic"

	"github.com/agubarev/hometown/pkg/client"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
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
	// DefaultIdleTimeout is the time since the last session activity
	DefaultIdleTimeout = timestamp.Timestamp(1e9 * 120) // 2 minutes

	// DefaultSessionTTL is the default session lifetime
	DefaultSessionTTL = timestamp.Timestamp(1e9 * 600) // 10 minutes
)

// IdentityKind represents an identity kind (i.e.: user/service/application)
type IdentityKind uint8

const (
	IKUser = iota
	IKServer
	IKApplication
	IKUnknown
)

func (k IdentityKind) String() string {
	switch k {
	case IKUser:
		return "user"
	case IKServer:
		return "server"
	case IKApplication:
		return "application"
	default:
		return "unrecognized identity"
	}
}

// Identity represents a session owner
type Identity struct {
	ID   uuid.UUID    `json:"id"`
	Kind IdentityKind `json:"kind"`
}

func UserIdentity(id uuid.UUID) Identity    { return Identity{ID: id, Kind: IKUser} }
func ServerIdentity(id uuid.UUID) Identity  { return Identity{ID: id, Kind: IKServer} }
func AppIdentity(id uuid.UUID) Identity     { return Identity{ID: id, Kind: IKApplication} }
func UnknownIdentity(id uuid.UUID) Identity { return Identity{ID: id, Kind: IKUnknown} }

// Owner represents a fusion of a remote client that acts
// at the behest of an identity (typically the end user)
type Owner struct {
	Client   client.Client `json:"client"`
	Identity Identity      `json:"id"`
}

func NewOwner(c client.Client, ident Identity) Owner {
	return Owner{
		Client:   c,
		Identity: ident,
	}
}

// Session represents an authenticated session
type Session struct {
	Owner Owner `db:"owner" json:"owner"`

	// ID is also used as JTI (JWT ID)
	ID uuid.UUID `db:"id" json:"id"`

	// UserAgent is the user agent taken from the client at the time of authentication
	UserAgent string `db:"user_agent" json:"user_agent"`

	// IP is the IP address from which this session has been initiated
	IP user.IPAddr `db:"ip" json:"ip"`

	// Flags describes metadata like whether it's idling, revoked,
	// revoked by its owner, expiry, client or some external system
	Flags uint32 `db:"flags" json:"flags"`

	// times
	CreatedAt    timestamp.Timestamp `db:"created_at" json:"created_at"`
	LastActiveAt timestamp.Timestamp `db:"last_active_at" json:"last_active_at"`
	RefreshedAt  timestamp.Timestamp `db:"refreshed_at" json:"refreshed_at"`
	RevokedAt    timestamp.Timestamp `db:"revoked_at" json:"revoked_at"`
	ExpireAt     timestamp.Timestamp `db:"expire_at" json:"expire_at"`
}

func NewSession(owner Owner, meta RequestMetadata, ttl timestamp.Timestamp) (s *Session, err error) {
	if ttl == 0 {
		ttl = DefaultSessionTTL
	}

	s = &Session{
		Owner:        owner,
		ID:           uuid.New(),
		UserAgent:    meta.UserAgent,
		IP:           meta.IP,
		Flags:        0,
		CreatedAt:    timestamp.Now(),
		LastActiveAt: 0,
		RefreshedAt:  0,
		RevokedAt:    0,
		ExpireAt:     timestamp.Now() + ttl,
	}

	return s, nil
}

// SanitizeAndValidate validates the session
func (s *Session) Validate() error {
	if s.ID == uuid.Nil {
		return ErrInvalidSessionID
	}

	if s.ExpireAt == 0 {
		return ErrZeroExpiration
	}

	if s.Owner.Identity.ID == uuid.Nil {
		return ErrInvalidIdentityID
	}

	return nil
}

// TimeLeft returns the time remaining before it expires
// NOTE: in nanoseconds
func (s *Session) TimeLeft() timestamp.Timestamp {
	ttl := int64(s.ExpireAt - timestamp.Now())
	if ttl < 0 {
		return timestamp.Timestamp(0)
	}

	return timestamp.Timestamp(ttl)
}

func (s *Session) revoke(rflags uint32) error {
	if s.RevokedAt != 0 {
		return ErrSessionAlreadyRevoked
	}

	if rflags&(SRevokedByClient|SRevokedByLogout|SRevokedBySystem|SRevokedByExpiry) == 0 {
		return ErrInvalidRevocationFlag
	}

	// setting revocation timestamp
	atomic.StoreUint64((*uint64)(&s.RevokedAt), uint64(timestamp.Now()))

	// updating revocation flags
	for {
		if atomic.CompareAndSwapUint32(&s.Flags, s.Flags, s.Flags|rflags) {
			return nil
		}
	}
}

func (s *Session) Touch()                       { atomic.StoreUint64((*uint64)(&s.LastActiveAt), uint64(timestamp.Now())) }
func (s *Session) RevokeByClient() error        { return s.revoke(SRevokedByClient) }
func (s *Session) RevokeBySystem() error        { return s.revoke(SRevokedBySystem) }
func (s *Session) RevokeByExpiry() error        { return s.revoke(SRevokedByExpiry) }
func (s *Session) IsIdle() bool                 { return s.LastActiveAt > DefaultIdleTimeout }
func (s *Session) IsValid() bool                { return s.RevokedAt == 0 && s.ExpireAt < timestamp.Now() }
func (s *Session) IsRevoked() bool              { return s.RevokedAt != 0 }
func (s *Session) IsExpired() bool              { return s.ExpireAt < timestamp.Now() }
func (s *Session) IsRevokedByLogout() bool      { return s.Flags&SRevokedByLogout == SRevokedByLogout }
func (s *Session) IsRevokedByExpiry() bool      { return s.Flags&SRevokedByExpiry == SRevokedByExpiry }
func (s *Session) IsRevokedByClient() bool      { return s.Flags&SRevokedByClient == SRevokedByClient }
func (s *Session) IsRevokedBySystem() bool      { return s.Flags&SRevokedBySystem == SRevokedBySystem }
func (s *Session) IsCreatedByCredentials() bool { return s.Flags&SCreatedByCreds == SCreatedByCreds }
func (s *Session) IsCreatedByRefToken() bool    { return s.Flags&SCreatedByRefToken == SCreatedByRefToken }
