package password_test

import (
	"crypto/rand"
	"testing"

	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestNewPassword(t *testing.T) {
	a := assert.New(t)

	correctPassword := []byte("1j20nmdoansd-[afkcq0ofecimwq1")
	wrongPassword := []byte("wrongpassword")
	p, err := password.New(password.OKUser, uuid.New(), correctPassword, []string{})
	a.NoError(err)
	a.NotNil(p)
	a.True(p.Compare(correctPassword))
	a.False(p.Compare(wrongPassword))
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
