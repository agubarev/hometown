package util

import (
	"encoding/hex"
	"math/rand"
)

// NewCSPRNG returns a slice of random bytes
func NewCSPRNG(nbytes int) ([]byte, error) {
	buf := make([]byte, nbytes)

	// will only checking for an error
	_, err := rand.Read(buf)
	if err != nil {
		return buf, err
	}

	return buf, err
}

// NewCSPRNGHex is a string wrapper for NewCSPRNG
func NewCSPRNGHex(nbytes int) (string, error) {
	bs, err := NewCSPRNG(nbytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bs), nil
}
