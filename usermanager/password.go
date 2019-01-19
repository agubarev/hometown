package usermanager

import (
	"github.com/oklog/ulid"
	"golang.org/x/crypto/bcrypt"
)

// Password object
type Password struct {
	// password ID must be equal to the user ID
	ID   ulid.ULID `json:"id"`
	Hash []byte    `json:"h"`
}

// NewPassword creates a hash from a given raw string
func NewPassword(id ulid.ULID, rawpass string) (*Password, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(rawpass), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	p := &Password{
		ID:   id,
		Hash: h,
	}

	return p, nil
}

// Compare tests whether a given plaintext password is valid
func (p *Password) Compare(rawpass string) bool {
	if err := bcrypt.CompareHashAndPassword(p.Hash, []byte(rawpass)); err == nil {
		return true
	}

	return false
}
