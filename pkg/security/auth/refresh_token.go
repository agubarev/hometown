package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/agubarev/hometown/pkg/client"
	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

const (
	Length = 32
	DefaultRefreshTokenTTL
)

// Hash represents the very secret essence of a refresh token
// TODO: implement hash encryption for the store
// TODO: implement hash signature and verification
type Hash [Length]byte

// EmptyHash is a predefined sample for comparison
var EmptyHash = Hash{}

func NewTokenHash() (hash Hash) {
	if _, err := rand.Read(hash[:]); err != nil {
		panic(errors.Wrap(err, "failed to generate refresh token hash"))
		return hash
	}

	return hash
}

func (h Hash) Validate() error {
	if h == EmptyHash {
		return ErrRefreshTokenIsEmpty
	}

	return nil
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

type RefreshToken struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	LastSessionID      uuid.UUID `db:"last_session_id" json:"last_session_id"`
	LastRefreshTokenID uuid.UUID `db:"prev_rtok_id" json:"prev_rtok_id"`
	ClientID           uuid.UUID `db:"client_id" json:"client_id"`
	Identity           Identity  `db:"identity" json:"identity"`
	Hash               Hash      `db:"hash" json:"hash"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	RotatedAt          time.Time `db:"rotated_at" json:"rotated_at"`
	ExpireAt           time.Time `db:"expire_at" json:"expire_at"`
	Flags              uint8     `db:"flags" json:"flags"`
	_                  struct{}
}

func (rtok *RefreshToken) IsExpired() bool { return rtok.ExpireAt.After(time.Now()) }
func (rtok *RefreshToken) IsRotated() bool { return !rtok.RotatedAt.IsZero() }
func (rtok *RefreshToken) IsActive() bool  { return !rtok.IsExpired() && !rtok.IsRotated() }

func NewRefreshToken(jti uuid.UUID, c *client.Client, identity Identity, expireAt time.Time) (rtok RefreshToken, err error) {
	// client must be given
	if c == nil {
		return rtok, client.ErrNilClient
	}

	// session (access token) ID must be provided
	if jti == uuid.Nil {
		return rtok, ErrInvalidJTI
	}

	// expiration time must be provided
	if expireAt.IsZero() {
		return rtok, ErrZeroExpiration
	}

	// and not in the past time
	if expireAt.Before(time.Now()) {
		return rtok, ErrInvalidExpirationTime
	}

	// it is unsafe to provide refresh tokens to clients
	// that are designated as non-confidential
	if !c.IsConfidential() {
		return rtok, ErrClientIsNonconfidential
	}

	rtok = RefreshToken{
		Hash:          NewTokenHash(),
		ClientID:      c.ID,
		Identity:      identity,
		ID:            uuid.New(),
		LastSessionID: jti,
		CreatedAt:     time.Now(),
		ExpireAt:      expireAt,
		Flags:         0,
	}

	return rtok, nil
}
