package group_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGroupNew(t *testing.T) {
	a := assert.New(t)

	g, err := group.NewGroup(group.FEnabled|group.FGroup, uuid.Nil, group.Key("test_key"), group.Name("test group name"))
	a.NoError(err)
	a.Equal(group.FGroup|group.FEnabled, g.Flags)
	a.True(g.IsEnabled())
	a.True(g.IsGroup())
	a.Equal(group.Key("test_key"), g.Key)
	a.Equal(group.Name("test group name"), g.DisplayName)
	a.False(g.IsDefault())
	a.True(g.IsEnabled())
}
