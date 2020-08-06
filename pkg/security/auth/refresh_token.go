package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"

	"github.com/agubarev/hometown/pkg/util/timestamp"
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

func NewHash() (hash Hash) {
	if _, err := rand.Read(hash[:]); err != nil {
		panic(errors.Wrap(err, "failed to generate refresh token hash"))
		return hash
	}

	return hash
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
	Token       Hash                `db:"token" json:"token"`
	Owner       Client              `db:"owner" json:"owner"`
	ID          uuid.UUID           `db:"id" json:"id"`
	LastTokenID uuid.UUID           `db:"last_token_id" json:"last_token_id"`
	CreatedAt   timestamp.Timestamp `db:"created_at" json:"created_at"`
	ExpireAt    timestamp.Timestamp `db:"expire_at" json:"expire_at"`
	Flags       uint8               `db:"flags" json:"flags"`
}

func NewRefreshToken()
