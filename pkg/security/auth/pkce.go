package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"unsafe"
)

type PKCEChallenge struct {
	Challenge string
	Method    string
}

func NewPKCEChallenge(challenge string, method string) (c PKCEChallenge, err error) {
	c = PKCEChallenge{
		Challenge: challenge,
		Method:    strings.ToLower(strings.TrimSpace(method)),
	}

	return c, c.Validate()
}

func (c *PKCEChallenge) Validate() error {
	if len(c.Challenge) == 0 {
		return ErrEmptyCodeChallenge
	}

	if c.Method == "" {
		return ErrEmptyCodeChallengeMethod
	}

	if c.Method != "s256" && c.Method != "plain" {
		return ErrInvalidCodeChallengeMethod
	}

	return nil
}

func (c *PKCEChallenge) Verify(verifier string) bool {
	switch c.Method {
	case "s256":
		s256checksum := sha256.Sum256(*(*[]byte)(unsafe.Pointer(&verifier)))
		return string(c.Challenge) == base64.URLEncoding.EncodeToString(s256checksum[:])
	case "plain":
		return string(c.Challenge) == verifier
	default:
		return false
	}
}
