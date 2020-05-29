package password_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/password"

	"github.com/stretchr/testify/assert"
)

func TestNewPassword(t *testing.T) {
	a := assert.New(t)

	correctPassword := []byte("1j20nmdoansd-[afkcq0ofecimwq1")
	wrongPassword := []byte("wrongpassword")
	p, err := password.New(password.KUser, 1, correctPassword, []string{})
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

	err = password.EvaluatePasswordStrength([]byte("jwfjowfjowjeofwjoefwjoefqjiqweoqpw[eofqwp-oefkqpwefoq"), []string{})
	a.Error(err)
	a.EqualError(password.ErrLongPassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("12345678"), []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("123Andrei9991superp@ss"), []string{})
	a.Error(err)
	a.EqualError(password.ErrUnsafePassword, err.Error())

	err = password.EvaluatePasswordStrength([]byte("s@fer!@()*!p@ssw0rd"), []string{})
	a.Error(err)
	a.NoError(err)
}
