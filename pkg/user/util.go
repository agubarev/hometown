package user

import (
	"bytes"
	"strings"
)

func UsernameFromString(src string) (dest TUsername) {
	copy(dest[:], strings.TrimSpace(strings.ToLower(src)))
	return dest
}

func UsernameFromBytes(src []byte) (dest TUsername) {
	copy(dest[:], bytes.TrimSpace(bytes.ToLower(src)))
	return dest
}

func TGroupNameFromString(src string) (dest TGroupName) {
	copy(dest[:], strings.TrimSpace(strings.ToLower(src)))
	return dest
}

func TGroupNameFromBytes(src []byte) (dest TGroupName) {
	copy(dest[:], bytes.TrimSpace(bytes.ToLower(src)))
	return dest
}
