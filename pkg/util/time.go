package util

import "time"

func NowUnixU32() uint32 {
	return uint32(time.Now().Unix())
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
