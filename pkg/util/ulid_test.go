package util_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestNewULID(t *testing.T) {
	a := assert.New(t)
	uid1 := util.NewULID()
	uid2 := util.NewULID()
	uid3 := util.NewULID()

	a.NotNil(uid1)
	a.NotNil(uid2)
	a.NotNil(uid3)

	a.NotEqual(uid1, uid2)
	a.NotEqual(uid2, uid3)
	a.NotEqual(uid3, uid1)
}
