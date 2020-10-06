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
	Hash          Hash      `db:"hash" json:"hash"`
	ClientID      uuid.Time `db:"client" json:"client"`
	Identity      Identity  `db:"identity" json:"identity"`
	ID            uuid.UUID `db:"id" json:"id"`
	LastSessionID uuid.UUID `db:"last_token_id" json:"last_token_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	ExpireAt      time.Time `db:"expire_at" json:"expire_at"`
	Flags         uint8     `db:"flags" json:"flags"`
}

func NewRefreshToken(
	jti uuid.UUID,
	c *client.Client,
	identity Identity,
	expireAt time.Time,
) (
	rt RefreshToken,
	err error,
) {
	if !c.IsConfidential() {
		return rt, ErrClientIsNonconfidential
	}

	rt = RefreshToken{
		Hash:          NewTokenHash(),
		Identity:      identity,
		ID:            uuid.New(),
		LastSessionID: jti,
		CreatedAt:     time.Now(),
		ExpireAt:      expireAt,
		Flags:         0,
	}

	return rt, nil
}

func (rtok *RefreshToken) IsExpired() bool {
	return rtok.ExpireAt.After(time.Now())
}
