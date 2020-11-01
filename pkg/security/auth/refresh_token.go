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
	Length                 = 32
	DefaultRefreshTokenTTL = 24 * time.Hour
)

// RefreshTokenHash represents the very secret essence of a refresh token
// TODO: implement hash encryption for the store
// TODO: implement hash signature and verification
type RefreshTokenHash [Length]byte

// EmptyRefreshTokenHash is a predefined sample for comparison
var EmptyRefreshTokenHash = RefreshTokenHash{}

func NewTokenHash() (hash RefreshTokenHash) {
	if _, err := rand.Read(hash[:]); err != nil {
		panic(errors.Wrap(err, "failed to generate refresh token hash"))
		return hash
	}

	return hash
}

func (h RefreshTokenHash) Validate() error {
	if h == EmptyRefreshTokenHash {
		return ErrRefreshTokenIsEmpty
	}

	return nil
}

func (h RefreshTokenHash) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if h[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(h[:], byte(0))
	if zpos == -1 {
		return append(buf, h[:]...), nil
	}

	return append(buf, h[0:zpos]...), nil
}

func (h RefreshTokenHash) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if src == nil {
		return nil
	}

	copy(h[:], src)

	return nil
}

func (h RefreshTokenHash) String() string {
	return hex.EncodeToString(h[:])
}

type RefreshToken struct {
	ID                     uuid.UUID        `db:"id" json:"id"`
	TraceID                uuid.UUID        `db:"trace_id" json:"trace_id"`
	ParentID               uuid.UUID        `db:"parent_id" json:"parent_id"`
	RotatedID              uuid.UUID        `db:"rotated_id" json:"rotated_id"`
	LastSessionID          uuid.UUID        `db:"last_session_id" json:"last_session_id"`
	PreviousRefreshTokenID uuid.UUID        `db:"previous_refresh_token_id" json:"previous_refresh_token_id"`
	ClientID               uuid.UUID        `db:"client_id" json:"client_id"`
	Identity               Identity         `db:"identity" json:"identity"`
	Hash                   RefreshTokenHash `db:"hash" json:"hash"`
	CreatedAt              time.Time        `db:"created_at" json:"created_at"`
	RotatedAt              time.Time        `db:"rotated_at" json:"rotated_at"`
	RevokedAt              time.Time        `db:"revoked_at" json:"revoked_at"`
	ExpireAt               time.Time        `db:"expire_at" json:"expire_at"`
	Flags                  uint8            `db:"flags" json:"flags"`
	_                      struct{}
}

func (rtok *RefreshToken) IsExpired() bool { return rtok.ExpireAt.Before(time.Now()) }
func (rtok *RefreshToken) IsRotated() bool { return !rtok.RotatedAt.IsZero() }
func (rtok *RefreshToken) IsRevoked() bool { return !rtok.RevokedAt.IsZero() }

func (rtok *RefreshToken) IsActive() (bool, error) {
	if rtok.IsExpired() {
		return false, ErrRefreshTokenExpired
	}

	if rtok.IsRotated() {
		return false, ErrRefreshTokenRotated
	}

	if rtok.IsRevoked() {
		return false, ErrRefreshTokenRevoked
	}

	return true, nil
}

func NewRefreshToken(
	traceID uuid.UUID,
	jti uuid.UUID,
	c *client.Client,
	identity Identity,
	expireAt time.Time,
) (rtok RefreshToken, err error) {
	// client must be given
	if c == nil {
		return rtok, client.ErrNilClient
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

	// trace ID must be given
	if traceID == uuid.Nil {
		return rtok, ErrInvalidTraceID
	}

	rtok = RefreshToken{
		Hash:          NewTokenHash(),
		TraceID:       traceID,
		ID:            uuid.New(),
		ClientID:      c.ID,
		Identity:      identity,
		LastSessionID: jti,
		CreatedAt:     time.Now(),
		ExpireAt:      expireAt,
		Flags:         0,
	}

	return rtok, rtok.Validate()
}

func NewRotatedRefreshToken(currentToken RefreshToken) (newToken RefreshToken, err error) {
	if err = currentToken.Validate(); err != nil {
		return newToken, errors.Wrap(err, "current refresh token validation has failed")
	}

	// checking the status of the current token
	if ok, err := currentToken.IsActive(); !ok {
		return newToken, errors.Wrap(err, "failed to initialize rotated refresh token due to a current token problem")
	}

	// re-usable timestamp
	now := time.Now()

	newToken = RefreshToken{
		Hash:          NewTokenHash(),
		ID:            uuid.New(),
		TraceID:       currentToken.TraceID,
		ParentID:      currentToken.ID,
		ClientID:      currentToken.ClientID,
		Identity:      currentToken.Identity,
		LastSessionID: currentToken.LastSessionID,
		CreatedAt:     now,
		ExpireAt:      now.Add(DefaultRefreshTokenTTL),
		Flags:         0,
	}

	return newToken, nil
}

func (rtok *RefreshToken) Validate() error {
	if rtok.ID == uuid.Nil {
		return ErrInvalidRefreshTokenID
	}

	if rtok.ClientID == uuid.Nil {
		return client.ErrInvalidClientID
	}

	if rtok.TraceID == uuid.Nil {
		return ErrInvalidTraceID
	}

	// session (access token) ID must be provided
	if rtok.LastSessionID == uuid.Nil {
		return ErrInvalidJTI
	}

	// expiration time must be provided
	if rtok.ExpireAt.IsZero() {
		return ErrZeroExpiration
	}

	return nil
}
