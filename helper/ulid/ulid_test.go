package ulid_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/helper/ulid"
)

func TestNewULID(t *testing.T) {
	a := assert.New(t)
	uid1 := ulid.NewULID()
	uid2 := ulid.NewULID()
	uid3 := ulid.NewULID()

	a.NotNil(uid1)
	a.NotNil(uid2)
	a.NotNil(uid3)

	a.NotEqual(uid1, uid2)
	a.NotEqual(uid2, uid3)
	a.NotEqual(uid3, uid1)
}
