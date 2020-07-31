package bytearray

import (
	"bytes"

	"github.com/jackc/pgx/pgtype"
)

type ByteString32 [32]byte

var NilByteString32 = ByteString32{}

func NewByteString32(s string) (bs ByteString32) {
	copy(bs[:], bytes.TrimSpace([]byte(s)))
	return bs
}

func (bs ByteString32) String() string {
	if bs[0] == 0 {
		return ""
	}

	zeroPos := bytes.IndexByte(bs[:], byte(0))
	if zeroPos == -1 {
		return string(bs[:])
	}

	return string(bs[0:zeroPos])
}

func (bs *ByteString32) Trim() {
	copy(bs[:], bytes.TrimSpace(bs[:]))
}

func (bs *ByteString32) ToLower() {
	copy(bs[:], bytes.ToLower(bs[:]))
}

func (bs ByteString32) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if bs[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(bs[:], byte(0))
	if zpos == -1 {
		return append(buf, bs[:]...), nil
	}

	return append(buf, bs[0:zpos]...), nil
}

func (bs *ByteString32) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	copy(bs[:], src)
	return nil
}
