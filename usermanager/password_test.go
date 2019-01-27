package usermanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluatePassword(t *testing.T) {
	a := assert.New(t)

	p := "1234567"
	err := evaluatePassword(p, []string{})
	a.EqualError(ErrShortPassword, err.Error())

	p = "jwfjowfjowjeofwjoefwjoefqjiqweoqpw[eofqwp-oefkqpwefoq"
	err = evaluatePassword(p, []string{})
	a.EqualError(ErrLongPassword, err.Error())

	p = "12345678"
	err = evaluatePassword(p, []string{})
	a.EqualError(ErrUnsafePassword, err.Error())

	p = "s@fer!@()*!p@ssw0rd"
	err = evaluatePassword(p, []string{})
	a.NoError(err)
}
