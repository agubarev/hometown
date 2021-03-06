package password

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	zxcvbn "github.com/nbutton23/zxcvbn-go"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

// retryAttempts defines how many generation attempts a password
// gets if it keeps failing strength test
const retryAttempts = 5

type GenFlags uint8

var charPool [4][]rune

func init() {
	for c := 'a'; c <= 'z'; c++ {
		charPool[0] = append(charPool[0], c)
	}

	for c := 'A'; c <= 'Z'; c++ {
		charPool[1] = append(charPool[1], c)
	}

	for c := '1'; c <= '9'; c++ {
		charPool[2] = append(charPool[2], c)
	}

	charPool[3] = []rune{
		'!', '@', '#', '$', '%', '^', '&', '*',
	}
}

const (
	GFNumber GenFlags = 1 << iota
	GFSpecial
	GFMixCase
	GFDefault = GFNumber | GFMixCase | GFSpecial
)

// constant rules
const (
	MinLength  = 8
	MaxLength  = 99
	DefaultTTL = 24 * 182 * time.Hour
)

type Owner struct {
	Kind Kind      `db:"kind" json:"kind"`
	ID   uuid.UUID `db:"id" json:"id"`
}

func NewOwner(k Kind, id uuid.UUID) Owner {
	return Owner{
		Kind: k,
		ID:   id,
	}
}

type Kind uint8

// password owner kinds
const (
	OKUser Kind = 1
	OKApplication
)

// Password object
// TODO: use byte array instead of slice for password hash
type Password struct {
	Owner
	Hash             []byte    `db:"hash" json:"-"`
	CreatedAt        time.Time `db:"created_at" json:"-"`
	UpdatedAt        time.Time `db:"updated_at" json:"-"`
	ExpireAt         time.Time `db:"expire_at" json:"-"`
	IsChangeRequired bool      `db:"is_change_required" json:"-"`
}

func generate(length int, pscore int, flags GenFlags) (raw []byte, err error) {
	if length < MinLength {
		return raw, ErrShortPassword
	}

	if length > MaxLength {
		return raw, ErrLongPassword
	}

	// raw password will be stored here
	raw = make([]byte, length)

	// character pool
	pool := make([]rune, 26)

	// copying base pool (lower-case alpha)
	copy(pool, charPool[0])

	switch true {
	case flags&GFMixCase == GFMixCase:
		pool = append(pool, charPool[1]...)
		fallthrough
	case flags&GFNumber == GFNumber:
		pool = append(pool, charPool[2]...)
		fallthrough
	case flags&GFSpecial == GFSpecial:
		pool = append(pool, charPool[3]...)
	}

	// current pool length
	plen := len(pool)

	// current retry attempts
	attempts := 0

	// generate password until it passes validation
	// NOTE: rewriting raw password characters on repetitive iterations
	for {
		// generation attempts
		attempts++

		for i := 0; i < length; i++ {
			raw[i] = byte(pool[rand.Intn(plen)])
		}

		// break out of the loop if there was no error
		if EvaluatePasswordStrength(raw, pscore, []string{}) == nil {
			break
		}

		// determining safety feasibility
		if attempts >= retryAttempts {
			return nil, ErrInfeasibleSafety
		}
	}

	return raw, nil
}

func New(owner Owner, length int, pscore int, flags GenFlags) (p Password, raw []byte, err error) {
	// generating raw password
	raw, err = generate(length, pscore, flags)
	if err != nil {
		return p, nil, errors.Wrap(err, "failed to generate password")
	}

	// generating password hash
	h, err := bcrypt.GenerateFromPassword(raw, bcrypt.DefaultCost)
	if err != nil {
		return p, raw, err
	}

	p = Password{
		Owner:            owner,
		Hash:             h,
		CreatedAt:        time.Now(),
		ExpireAt:         time.Now().Add(DefaultTTL),
		IsChangeRequired: false,
	}

	// recur if validation fails
	if err = p.Validate(); err != nil {
		return New(owner, length, 3, flags)
	}

	return p, raw, nil
}

func NewRaw(length int, pscore int, flags GenFlags) (raw []byte) {
	raw, err := generate(length, pscore, flags)
	if err != nil {
		panic(errors.Wrap(err, "failed to generate raw password"))
	}

	return raw
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
func EvaluatePasswordStrength(rawpass []byte, pscore int, data []string) error {
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
	if result.Score < pscore {
		return ErrUnsafePassword
	}

	return nil
}

// NewFromInput creates a hash from a given raw password byte slice
func NewFromInput(o Owner, rawpass []byte, data []string) (p Password, err error) {
	if err = EvaluatePasswordStrength(rawpass, 3, data); err != nil {
		return p, err
	}

	h, err := bcrypt.GenerateFromPassword(rawpass, bcrypt.DefaultCost)
	if err != nil {
		return p, err
	}

	p = Password{
		Owner:     o,
		Hash:      h,
		CreatedAt: time.Now(),
		ExpireAt:  time.Now().Add(DefaultTTL),
	}

	return p, nil
}

// Compare tests whether a given plaintext password is valid
func (p Password) Compare(rawpass []byte) bool {
	return bcrypt.CompareHashAndPassword(p.Hash, rawpass) == nil
}
