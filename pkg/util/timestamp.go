package util

import (
	"encoding/binary"
	"time"

	"github.com/jackc/pgx/pgio"
	"github.com/jackc/pgx/pgtype"
)

type Timestamp uint32

func NewTimestampFromUnix(ts uint32) Timestamp {
	return Timestamp(ts)
}

func NowUnixU32() Timestamp {
	return Timestamp(time.Now().Unix())
}

func NowUnixI64() int64 {
	return time.Now().Unix()
}

func TimeFromU32Unix(ts uint32) time.Time {
	return time.Unix(int64(ts), 0)
}

func TimeFromI64Unix(ts int64) time.Time {
	return time.Unix(ts, 0)
}

func (ts Timestamp) String() string {
	return time.Unix(int64(ts), 0).String()
}

func (ts Timestamp) EncodeBinary(ci *pgtype.ConnInfo, buf []byte) (newBuf []byte, err error) {
	if ts == 0 {
		return nil, nil
	}

	return pgio.AppendInt64(buf, int64(ts)*1e6), nil
}

func (ts *Timestamp) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	*ts = Timestamp(binary.BigEndian.Uint64(src) / 1e6)
	return nil
}

func (ts Timestamp) Time() time.Time {
	return time.Unix(int64(ts), 0)
}
