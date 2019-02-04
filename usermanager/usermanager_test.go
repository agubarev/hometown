package usermanager_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
)

func TestUserManagerTestNew(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	m, err := usermanager.New()
	a.NoError(err)
	a.NotNil(m)

	// TODO: check existence of all necessary paths

	err = m.Init()
	a.NoError(err)
}
