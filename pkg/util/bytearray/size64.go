package bytearray

import (
	"bytes"

	"github.com/jackc/pgx/pgtype"
)

type ByteString64 [64]byte

func NewByteString64(s string) (bs ByteString64) {
	copy(bs[:], bytes.ToLower(bytes.TrimSpace([]byte(s))))
	return bs
}

var NilByteString64 = ByteString64{}

func (bs ByteString64) String() string {
	if bs[0] == 0 {
		return ""
	}

	zeroPos := bytes.IndexByte(bs[:], byte(0))
	if zeroPos == -1 {
		return string(bs[:])
	}

	return string(bs[0:zeroPos])
}

func (bs *ByteString64) Trim() {
	copy(bs[:], bytes.TrimSpace(bs[:]))
}

func (bs *ByteString64) ToLower() {
	copy(bs[:], bytes.ToLower(bs[:]))
}

func (bs ByteString64) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if bs[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(bs[:], byte(0))
	if zpos == -1 {
		return append(buf, bs[:]...), nil
	}

	return append(buf, bs[0:zpos]...), nil
}

func (bs *ByteString64) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	copy(bs[:], src)
	return nil
}
