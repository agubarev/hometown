package timestamp

import (
	"encoding/binary"
	"time"

	"github.com/jackc/pgx/pgio"
	"github.com/jackc/pgx/pgtype"
)

type Timestamp uint64

// microsecFromUnixEpochToY2K borrowed from pgx (pgtype) package
const microsecFromUnixEpochToY2K = 946684800 * 1000000

// nsecMask borrowed from time.Time
const nsecMask = 1<<30 - 1

func NewTimestampFromUnix(uts int64) Timestamp {
	return Timestamp(uts)
}

func Now() Timestamp {
	return Timestamp(time.Now().UnixNano())
}

func (ts Timestamp) String() string {
	return time.Unix(int64(ts), int64(ts&nsecMask)).String()
}

func (ts Timestamp) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if ts == 0 {
		return nil, nil
	}

	return pgio.AppendInt64(buf, int64(ts/1000-microsecFromUnixEpochToY2K)), nil
}

func (ts *Timestamp) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if src == nil {
		return nil
	}

	*ts = Timestamp(binary.BigEndian.Uint64(src))

	return nil
}

func (ts Timestamp) Time() time.Time {
	return time.Unix(int64(ts/1e9), int64(ts&nsecMask))
}

func (ts Timestamp) EncodeText(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if ts == 0 {
		return nil, nil
	}

	return pgio.AppendInt64(buf, int64(ts/1000-microsecFromUnixEpochToY2K)), nil
}

func (ts *Timestamp) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if src == nil {
		return nil
	}

	*ts = Timestamp(binary.BigEndian.Uint64(src))

	return nil
}

func (ts *Timestamp) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	*ts = Timestamp(binary.BigEndian.Uint64(src.([]byte)))

	return nil
}
