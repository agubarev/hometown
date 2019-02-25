package usermanager

import (
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"github.com/oklog/ulid"
	"golang.org/x/crypto/bcrypt"
)

const (
	PasswordMinLength = 8
	PasswordMaxLength = 50
)

// Password object
type Password struct {
	// password ID must be equal to the user ID
	ID   ulid.ULID `json:"id"`
	Hash []byte    `json:"h"`
}

// EvaluatePasswordStrength evaluates password's strength by checking length,
// complexity, characters used etc.
func EvaluatePasswordStrength(rawpass string, userInputs []string) error {
	pl := len(rawpass)
	if pl < PasswordMinLength {
		return ErrShortPassword
	}

	if pl > PasswordMaxLength {
		return ErrLongPassword
	}

	// evaluating password's strength by the library's score
	// the score must be at least 3
	result := zxcvbn.PasswordStrength(rawpass, userInputs)
	if result.Score < 3 {
		return ErrUnsafePassword
	}

	return nil
}

// NewPassword creates a hash from a given raw string
func NewPassword(id ulid.ULID, rawpass string, userInput []string) (*Password, error) {
	err := EvaluatePasswordStrength(rawpass, userInput)
	if err != nil {
		return nil, err
	}

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
