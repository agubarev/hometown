package bytearray

import (
	"bytes"
	"encoding/json"

	"github.com/jackc/pgx/pgtype"
)

type ByteString512 [512]byte

func NewByteString512(s string) (bs ByteString512) {
	copy(bs[:], bytes.TrimSpace([]byte(s)))
	return bs
}

var NilByteString512 = ByteString512{}

func (bs ByteString512) String() string {
	if bs[0] == 0 {
		return ""
	}

	zeroPos := bytes.IndexByte(bs[:], byte(0))
	if zeroPos == -1 {
		return string(bs[:])
	}

	return string(bs[0:zeroPos])
}

func (bs *ByteString512) Trim() {
	copy(bs[:], bytes.TrimSpace(bs[:]))
}

func (bs *ByteString512) ToLower() {
	copy(bs[:], bytes.ToLower(bs[:]))
}

func (bs ByteString512) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if bs[0] == 0 {
		return nil, nil
	}

	zpos := bytes.IndexByte(bs[:], byte(0))
	if zpos == -1 {
		return append(buf, bs[:]...), nil
	}

	return append(buf, bs[0:zpos]...), nil
}

func (bs *ByteString512) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	copy(bs[:], src)
	return nil
}

func (bs ByteString512) MarshalJSON() ([]byte, error) {
	return json.Marshal(bs.String())
}

func (bs *ByteString512) UnmarshalJSON(data []byte) error {
	copy(bs[:], bytes.Trim(data, "\\\" "))
	return nil
}
