package password

import (
	"time"

	"github.com/agubarev/hometown/pkg/util/timestamp"
	"github.com/google/uuid"
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"
)

type GenFlags uint8

var charPool []rune

func init() {
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
	panic("finish this")
	for c := 'a'; c <= 'z'; c++ {
		charPool = append(charPool, c)
	}
}

const (
	GFNumber GenFlags = 1 << iota
	GFSpecial
	GFMixCase
)

// constant rules
const (
	MinLength  = 8
	MaxLength  = 64
	DefaultTTL = 24 * 182 * time.Hour
)

type Owner struct {
	ID   uuid.UUID `db:"id" json:"id"`
	Kind Kind      `db:"kind" json:"kind"`
}

type Kind uint8

// password owner kinds
const (
	OKUser Kind = 1
)

// Password object
// TODO: use byte array instead of slice for password hash
type Password struct {
	Owner
	Hash             []byte              `db:"hash" json:"-"`
	CreatedAt        timestamp.Timestamp `db:"created_at" json:"-"`
	UpdatedAt        timestamp.Timestamp `db:"updated_at" json:"-"`
	ExpireAt         timestamp.Timestamp `db:"expire_at" json:"-"`
	IsChangeRequired bool                `db:"is_change_required" json:"-"`
}

func New(o Owner, l int, f GenFlags) (p Password, raw []byte) {
	raw = make([]byte, l)

	p = Password{
		Owner:            o,
		Hash:             nil,
		CreatedAt:        0,
		UpdatedAt:        0,
		ExpireAt:         0,
		IsChangeRequired: false,
	}

	return p, raw
}

// SanitizeAndValidate validates password
func (p Password) Validate() error {
	if p.Kind == 0 {
		return ErrZeroKind
	}

	if p.Owner.ID == uuid.Nil {
		return ErrNilOwnerID
	}

	if len(p.Hash) == 0 {
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
	if result.Score < 4 {
		return ErrUnsafePassword
	}

	return nil
}

// NewFromInput creates a hash from a given raw password byte slice
func NewFromInput(o Owner, rawpass []byte, data []string) (p Password, err error) {
	if err = EvaluatePasswordStrength(rawpass, data); err != nil {
		return p, err
	}

	h, err := bcrypt.GenerateFromPassword(rawpass, bcrypt.DefaultCost)
	if err != nil {
		return p, err
	}

	p = Password{
		Owner:     o,
		Hash:      h,
		CreatedAt: timestamp.Now(),
		ExpireAt:  timestamp.Timestamp(time.Now().Add(DefaultTTL).Unix()),
	}

	return p, nil
}

// Compare tests whether a given plaintext password is valid
func (p Password) Compare(rawpass []byte) bool {
	return bcrypt.CompareHashAndPassword(p.Hash, rawpass) == nil
}
