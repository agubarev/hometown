package password_test

import (
	"crypto/rand"
	"testing"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestPassword(t *testing.T) {
	a := assert.New(t)

	correctPassword := []byte("1j20nmdoansd-[afkcq0ofecimwq1")
	wrongPassword := []byte("wrongpassword")

	o := password.Owner{
		ID:   uuid.New(),
		Kind: password.OKUser,
	}

	p, err := password.NewFromInput(o, correctPassword, []string{})
	a.NoError(err)
	a.NotNil(p)
	a.True(p.Compare(correctPassword))
	a.False(p.Compare(wrongPassword))
}

func TestNew(t *testing.T) {
	a := assert.New(t)

	o := password.Owner{
		ID:   uuid.New(),
		Kind: password.OKUser,
	}

	// alpha
	p, raw, err := password.New(o, 8, 3, 0)
	a.NoError(err)
	a.Equal(o, p.Owner)
	a.NotEmpty(raw)

	// alpha, mixed
	p, raw, err = password.New(o, 8, 3, password.GFMixCase)
	a.NoError(err)
	a.Equal(o, p.Owner)
	a.NotEmpty(raw)

	// alpha, mixed, number
	p, raw, err = password.New(o, 8, 3, password.GFMixCase|password.GFNumber)
	a.NoError(err)
	a.Equal(o, p.Owner)
	a.NotEmpty(raw)

	// alpha, mixed, number, special
	p, raw, err = password.New(o, 8, 3, password.GFMixCase|password.GFNumber|password.GFSpecial)
	a.NoError(err)
	a.Equal(o, p.Owner)
	a.NotEmpty(raw)

	// checking safety feasibility (length: 8, passing score: 4)
	p, raw, err = password.New(o, 8, 4, 0)
	a.Error(err)
	a.EqualError(password.ErrInfeasibleSafety, err.Error())
}

func TestEvaluatePassword(t *testing.T) {
	a := assert.New(t)

	err := password.EvaluatePasswordStrength([]byte("1234567"), 3, []string{})
	a.Error(err)
	a.EqualError(password.ErrShortPassword, err.Error())

	// generating password which must be lenghtier than max allowed
	pass := make([]byte, password.MaxLength+1)
	_, err = rand.Read(pass)
	a.NoError(err)

	err = password.EvaluatePasswordStrength(pass, 3, []string{})
	a.Error(err)
	a.EqualError(password.ErrLongPassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("12345678"), 3, []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("123Andrei9991superp@ss"), 3, []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("s@fer!@()*!p@ssw0rd*!jahaajk8!*@^%"), 3, []string{})
	a.NoError(err)
}
