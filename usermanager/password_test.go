package usermanager_test

import (
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"
	"gitlab.com/agubarev/hometown/util"

	"github.com/stretchr/testify/assert"
)

func TestNewPassword(t *testing.T) {
	a := assert.New(t)

	correctPassword := "1j20nmdoansd-[afkcq0ofecimwq1"
	wrongPassword := "wrongpassword"
	p, err := usermanager.NewPassword(util.NewULID(), correctPassword, []string{})
	a.NoError(err)
	a.NotNil(p)
	a.True(p.Compare(correctPassword))
	a.False(p.Compare(wrongPassword))
}

func TestEvaluatePassword(t *testing.T) {
	a := assert.New(t)

	p := "1234567"
	ui := []string{}
	err := usermanager.EvaluatePasswordStrength(p, ui)
	a.EqualError(usermanager.ErrShortPassword, err.Error())

	p = "jwfjowfjowjeofwjoefwjoefqjiqweoqpw[eofqwp-oefkqpwefoq"
	ui = []string{}
	err = usermanager.EvaluatePasswordStrength(p, ui)
	a.EqualError(usermanager.ErrLongPassword, err.Error())

	p = "12345678"
	ui = []string{}
	err = usermanager.EvaluatePasswordStrength(p, ui)
	a.EqualError(usermanager.ErrUnsafePassword, err.Error())

	p = "123Andrei9991superp@ss"
	ui = []string{"Andrei", "Gubarev"}
	err = usermanager.EvaluatePasswordStrength(p, ui)
	a.EqualError(usermanager.ErrUnsafePassword, err.Error())

	p = "s@fer!@()*!p@ssw0rd"
	ui = []string{}
	err = usermanager.EvaluatePasswordStrength(p, ui)
	a.NoError(err)
}
