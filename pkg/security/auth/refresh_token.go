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

const Length = 32

// Hash represents the very secret essence of a refresh token
// TODO: implement hash encryption for the store
// TODO: implement hash signature and verification
type Hash [Length]byte

type RefreshToken struct {
	Token     [Length]byte `db:"token" json:"token"`
	Client    Client       `db:"client" json:"client"`
	ID        uuid.UUID
	Flags     uint8
	CreatedAt timestamp.Timestamp
	ExpireAt  timestamp.Timestamp
}

func NewRefreshToken() (rtok RefreshToken) {
	if _, err := rand.Read(rtok[:]); err != nil {
		panic(errors.Wrap(err, "failed to generate refresh token"))
		return rtok
	}

	return rtok
}

func (h RefreshToken) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if h[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(h[:], byte(0))
	if zpos == -1 {
		return append(buf, h[:]...), nil
	}

	return append(buf, h[0:zpos]...), nil
}

func (h RefreshToken) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if src == nil {
		return nil
	}

	copy(h[:], src)

	return nil
}

func (h RefreshToken) String() string {
	return hex.EncodeToString(h[:])
}
