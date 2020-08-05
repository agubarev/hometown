package auth

import (
	"time"

	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// TODO: consider storing some client fingerprint inside the session

// IdentityKind represents an identity kind (i.e.: user/service/application)
type IdentityKind uint8

const (
	IKUser = iota
	IKService
	IKApplication
)

func (k IdentityKind) String() string {
	switch k {
	case IKUser:
		return "user"
	case IKService:
		return "service"
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

func UserIdentity(id uuid.UUID) Identity {
	return Identity{
		ID:   id,
		Kind: IKUser,
	}
}

func ServiceIdentity(id uuid.UUID) Identity {
	return Identity{
		ID:   id,
		Kind: IKService,
	}
}

func AppIdentity(id uuid.UUID) Identity {
	return Identity{
		ID:   id,
		Kind: IKApplication,
	}
}

// Session represents an authenticated session
type Session struct {
	Owner Identity `db:"owner" json:"owner"`

	// ID is also used as JTI (JWT ID)
	ID uuid.UUID `db:"id" json:"id"`

	// UserAgent is the user agent taken from the client at the time of authentication
	UserAgent bytearray.ByteString64 `db:"user_agent" json:"user_agent"`

	// IP is the IP address from which this session has been initiated
	IP user.IPAddr `db:"ip" json:"ip"`

	// times
	CreatedAt   timestamp.Timestamp `db:"created_at" json:"created_at"`
	RefreshedAt timestamp.Timestamp `db:"refreshed_at" json:"refreshed_at"`
	RevokedAt   timestamp.Timestamp `db:"revoked_at" json:"revoked_at"`
	ExpireAt    timestamp.Timestamp `db:"expire_at" json:"expire_at"`
}

func NewSession(jti uuid.UUID, ident Identity, agent bytearray.ByteString64, ip user.IPAddr, ttl time.Duration) (s Session, err error) {
	s = Session{
		Owner:       ident,
		ID:          jti,
		UserAgent:   agent,
		IP:          ip,
		CreatedAt:   timestamp.Now(),
		RefreshedAt: 0,
		RevokedAt:   0,
		ExpireAt:    timestamp.Timestamp(ttl.Nanoseconds()),
	}

	return s, nil
}

// SanitizeAndValidate validates the session
func (s *Session) Validate() error {
	if s.ID == uuid.Nil {
		return errors.New("session id not set")
	}

	if s.ExpireAt == 0 {
		return errors.New("expiration time is not set")
	}

	if s.ExpireAt < timestamp.Now() {
		return ErrTokenExpired
	}

	return nil
}
