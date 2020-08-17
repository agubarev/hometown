package password_test

import (
	"crypto/rand"
	"testing"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/davecgh/go-spew/spew"
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

	p, raw := password.New(o, 32, password.GFNumber)
	a.Equal(o, p.Owner)
	a.NotEmpty(raw)
}

func TestEvaluatePassword(t *testing.T) {
	a := assert.New(t)

	err := password.EvaluatePasswordStrength([]byte("1234567"), []string{})
	a.Error(err)
	a.EqualError(password.ErrShortPassword, err.Error())

	// generating password which must be lenghtier than max allowed
	pass := make([]byte, password.MaxLength+1)
	_, err = rand.Read(pass)
	a.NoError(err)

	err = password.EvaluatePasswordStrength(pass, []string{})
	a.Error(err)
	a.EqualError(password.ErrLongPassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("12345678"), []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("123Andrei9991superp@ss"), []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("s@fer!@()*!p@ssw0rd*!jahaajk8!*@^%"), []string{})
	a.NoError(err)
}
