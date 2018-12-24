package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/user"
)

func TestNewUser(t *testing.T) {
	a := assert.New(t)
	u := user.NewUser("testuser", "test@example.com")
	a.NotNil(u)
}
