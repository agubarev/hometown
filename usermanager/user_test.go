package usermanager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
)

func TestNewUser(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	u, err := usermanager.NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
}

func BenchmarkCreateUsers(t *testing.B) {
	for i := 0; i <= t.N; i++ {

	}
}
