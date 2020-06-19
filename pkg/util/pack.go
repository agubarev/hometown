package util

// L32 is the leftmost 32 bits set to 1s
const L32 = (^uint64(0) >> 32) << 32

// R32 is the rightmost 32 bits set to 1s
const R32 = ^uint64(0) >> 32

func PackU32s(a, b uint32) uint64 {
	return (uint64(a) << 32) | uint64(b)
}

func UnpackU32s(n uint64) (result [2]uint32) {
	result[0] = uint32((n & L32) >> 32)
	result[1] = uint32(n & R32)

	return result
}
