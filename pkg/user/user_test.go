package user_test

import (
	"testing"

	"github.com/agubarev/hometown/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestNewUser(t *testing.T) {

	a := assert.New(t)
	u, err := user.UserNew("testuser", "test@example.com", map[string]string{
		"firstname": "Andrei",
		"lastname":  "Gubarev",
	})
	a.NoError(err)
	a.NotNil(u)
}

func TestIsRegisteredAndStored(t *testing.T) {

}
