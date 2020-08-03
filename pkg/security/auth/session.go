package auth

import (
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type SessionOwner uint8

const (
	DGeneric SessionOwner = 0
	DAdmin   SessionOwner = 1 << (iota - SessionOwner(1))
)

func (k SessionOwner) String() string {
	switch k {
	case CKUser:
		return "user"
	default:
		return "unrecognized session owner kind"
	}
}

// Session represents a user session
// NOTE: the session is used only to identify the session owner (user),
// verify the user's IPAddr and UserAgent, and when to expire
// WARNING: session object must never be shared with the client,
// because it contains the refresh token
type Session struct {
	// ID is also used as JTI (JWT ID)
	ID uuid.UUID `db:"id" json:"id"`

	// UserAgent is the user agent taken from the client at the time of authentication
	UserAgent bytearray.ByteString64 `db:"user_agent" json:"user_agent"`

	// RefreshToken is generated specifically for this session
	RefreshToken RefreshToken `db:"refresh_token" json:"refresh_token"`

	// UserID is the ID of a user that owns this session
	UserID uuid.UUID `db:"user_id" json:"user_id"`

	// IP is the IP address from which this session has been initiated
	IP user.IPAddr `db:"ip" json:"ip"`

	// times
	CreatedAt   timestamp.Timestamp `db:"created_at" json:"created_at"`
	RefreshedAt timestamp.Timestamp `db:"refreshed_at" json:"refreshed_at"`
	RevokedAt   timestamp.Timestamp `db:"revoked_at" json:"revoked_at"`
	ExpireAt    timestamp.Timestamp `db:"expire_at" json:"expire_at"`
}

func NewSession(userID uuid.UUID, agent bytearray.ByteString64, ip user.IPAddr) (s Session, err error) {

	return s, nil
}

// SanitizeAndValidate validates the session
func (s *Session) Validate() error {
	if s.ID == uuid.Nil {
		return errors.New("session id not set")
	}

	if s.UserID == uuid.Nil {
		return errors.New("user id is not set")
	}

	if s.ExpireAt == 0 {
		return errors.New("expiration time is not set")
	}

	if s.ExpireAt < timestamp.Now() {
		return ErrTokenExpired
	}

	return nil
}
