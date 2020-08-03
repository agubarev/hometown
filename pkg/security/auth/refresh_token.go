package auth

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"

	"github.com/jackc/pgx/pgtype"
	"github.com/pkg/errors"
)

const Length = 32

type RefreshToken [Length]byte

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
