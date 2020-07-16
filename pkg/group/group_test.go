package group_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/stretchr/testify/assert"
)

func TestGroupNew(t *testing.T) {
	a := assert.New(t)

	g, err := group.NewGroup(group.FGroup, 0, group.NewKey("test_key"), group.NewName("test group name"), false)
	a.NoError(err)
	a.Equal(group.FGroup, g.Flags)
	a.Equal(group.NewKey("test_key"), g.Key)
	a.Equal(group.NewName("test group name"), g.DisplayName)
	a.Equal(false, g.IsDefault)
}
