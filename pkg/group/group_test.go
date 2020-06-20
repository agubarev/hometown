package group_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/stretchr/testify/assert"
)

func TestGroupNew(t *testing.T) {
	a := assert.New(t)

	g, err := group.NewGroup(group.GKGroup, 0, group.NewKey("test_key"), group.NewName("test group name"), false)
	a.NoError(err)
	a.Equal(group.GKGroup, g.Kind)
	a.Equal(group.NewKey("test_key"), g.Key)
	a.Equal(group.NewName("test group name"), g.Name)
	a.Equal(false, g.IsDefault)
}
