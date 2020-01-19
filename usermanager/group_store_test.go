package usermanager_test

import (
	"testing"

	"github.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
)

func TestGroupStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	g, err = s.Put(g)
	a.NoError(err)
	a.NotNil(g)
}

func TestGroupStoreGet(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	g, err = s.Put(g)
	a.NoError(err)
	a.NotNil(g)

	fg, err := s.Fetch(g.ID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ID, fg.ID)
	a.Equal(g.Kind, fg.Kind)
	a.Equal(g.Key, fg.Key)
	a.Equal(g.Name, fg.Name)
}

func TestGroupStoreGetAll(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g1, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g1)
	a.NoError(err)

	g2, err := usermanager.NewGroup(usermanager.GKRole, "test_role", "test role", nil)
	a.NotNil(g2)
	a.NoError(err)

	g3, err := usermanager.NewGroup(usermanager.GKGroup, "test_group123", "test group 123", nil)
	a.NotNil(g3)
	a.NoError(err)

	g1, err = s.Put(g1)
	a.NoError(err)

	g2, err = s.Put(g2)
	a.NoError(err)

	g3, err = s.Put(g3)
	a.NoError(err)

	gs, err := s.FetchAllGroups()
	a.NoError(err)
	a.Len(gs, 3)

	a.Equal(g1.ID, gs[0].ID)
	a.Equal(g1.Kind, gs[0].Kind)
	a.Equal(g1.Key, gs[0].Key)
	a.Equal(g1.Name, gs[0].Name)

	a.Equal(g2.ID, gs[1].ID)
	a.Equal(g2.Kind, gs[1].Kind)
	a.Equal(g2.Key, gs[1].Key)
	a.Equal(g2.Name, gs[1].Name)

	a.Equal(g3.ID, gs[2].ID)
	a.Equal(g3.Kind, gs[2].Kind)
	a.Equal(g3.Key, gs[2].Key)
	a.Equal(g3.Name, gs[2].Name)
}

func TestGroupStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	g, err = s.Put(g)
	a.NoError(err)

	fg, err := s.Fetch(g.ID)
	a.NotNil(fg)
	a.NoError(err)

	err = s.Delete(g.ID)
	a.NoError(err)

	fg, err = s.Fetch(g.ID)
	a.Nil(fg)
	a.Error(err)
	a.EqualError(err, usermanager.ErrGroupNotFound.Error())
}

func TestGroupStoreRelations(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	u, err := usermanager.NewUser("testuser", "testuser@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u)

	// making sure there is no previous relation
	ok, err := s.HasRelation(g.ID, u.ID)
	a.NoError(err)
	a.False(ok)

	// creating a relation
	a.NoError(s.PutRelation(g.ID, u.ID))

	// now they must be related
	ok, err = s.HasRelation(g.ID, u.ID)
	a.NoError(err)
	a.True(ok)

	// breaking relation
	a.NoError(s.DeleteRelation(g.ID, u.ID))

	// making sure the relation is gone
	ok, err = s.HasRelation(g.ID, u.ID)
	a.NoError(err)
	a.False(ok)
}
