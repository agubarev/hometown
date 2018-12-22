package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {
	a := assert.New(t)
	u := NewUser("testuser", "test@example.com")
	a.NotNil(u)
}
