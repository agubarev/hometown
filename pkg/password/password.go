package password

import (
	"time"

	"github.com/gocraft/dbr/v2"
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"
)

// constant rules
const (
	MinLength = 8
	MaxLength = 64
)

// Password object
type Password struct {
	Kind             Kind         `db:"kind" json:"-"`
	OwnerID          int          `db:"owner_id" json:"-"`
	Hash             [60]byte     `db:"hash" json:"-"`
	CreatedAt        dbr.NullTime `db:"created_at" json:"-"`
	ExpireAt         dbr.NullTime `db:"expire_at" json:"-"`
	IsChangeRequired bool         `db:"is_change_required" json:"-"`
}

// Validate validates password
func (p *Password) Validate() error {
	if p.Kind == 0 {
		return ErrZeroKind
	}

	if p.OwnerID == 0 {
		return ErrZeroOwnerID
	}

	if p.Hash[0] == 0 {
		return ErrEmptyPassword
	}

	return nil
}

// EvaluatePasswordStrength evaluates password's strength by checking length,
// complexity, characters used etc.
func EvaluatePasswordStrength(rawpass []byte, data []string) error {
	pl := len(rawpass)
	if pl < MinLength {
		return ErrShortPassword
	}

	if pl > MaxLength {
		return ErrLongPassword
	}

	// evaluating password's strength by the library's score
	// the score must be at least 3
	result := zxcvbn.PasswordStrength(string(rawpass), data)
	if result.Score < 3 {
		return ErrUnsafePassword
	}

	return nil
}

// New creates a hash from a given raw password byte slice
func New(rawpass []byte, data []string) (*Password, error) {
	err := EvaluatePasswordStrength(rawpass, data)
	if err != nil {
		return nil, err
	}

	h, err := bcrypt.GenerateFromPassword(rawpass, bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	p := &Password{
		CreatedAt: dbr.NewNullTime(time.Now()),
	}

	copy(p.Hash[:], h)

	return p, nil
}

// Compare tests whether a given plaintext password is valid
func (p *Password) Compare(rawpass string) bool {
	if err := bcrypt.CompareHashAndPassword(p.Hash[:], []byte(rawpass)); err == nil {
		return true
	}

	return false
}
