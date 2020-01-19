package usermanager

import (
	"time"

	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"
)

// constant rules
const (
	PasswordMinLength = 8
	PasswordMaxLength = 50
)

// Password object
type Password struct {
	// password ID must be equal to the user ID
	OwnerID          int64     `json:"-" db:"owner_id"`
	Hash             []byte    `json:"-" db:"hash"`
	CreatedAt        time.Time `json:"-" db:"created_at"`
	IsChangeRequired bool      `json:"-" db:"is_change_req"`
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
		return ErrWeakPassword
	}

	return nil
}

// NewPassword creates a hash from a given raw string
func NewPassword(ownerID int64, rawpass string, userInput []string) (*Password, error) {
	err := EvaluatePasswordStrength(rawpass, userInput)
	if err != nil {
		return nil, err
	}

	h, err := bcrypt.GenerateFromPassword([]byte(rawpass), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	p := &Password{
		OwnerID:   ownerID,
		Hash:      h,
		CreatedAt: time.Now(),
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
