package bytearray

import (
	"bytes"

	"github.com/jackc/pgx/pgtype"
)

type ByteString16 [16]byte

func NewByteString16(s string) (bs ByteString16) {
	copy(bs[:], bytes.ToLower(bytes.TrimSpace([]byte(s))))
	return bs
}

func (bs ByteString16) String() string {
	if bs[0] == 0 {
		return ""
	}

	zeroPos := bytes.IndexByte(bs[:], byte(0))
	if zeroPos == -1 {
		return string(bs[:])
	}

	return string(bs[0:zeroPos])
}

func (bs ByteString16) Trim() {
	copy(bs[:], bytes.TrimSpace(bs[:]))
}

func (bs ByteString16) ToLower() {
	copy(bs[:], bytes.ToLower(bs[:]))
}

func (bs ByteString16) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	zpos := bytes.IndexByte(bs[:], byte(0))
	if zpos == -1 {
		return append(buf, bs[:]...), nil
	}

	return append(buf, bs[0:zpos]...), nil
}

func (bs ByteString16) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	copy(bs[:], src)
	return nil
}
