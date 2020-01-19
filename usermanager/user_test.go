package usermanager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/agubarev/hometown/usermanager"
)

func TestNewUser(t *testing.T) {
	
	a := assert.New(t)
	u, err := usermanager.NewUser("testuser", "test@example.com", map[string]string{
		"firstname": "Andrei",
		"lastname":  "Gubarev",
	})
	a.NoError(err)
	a.NotNil(u)
}

func TestIsRegisteredAndStored(t *testing.T) {

}
