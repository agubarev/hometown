package usermanager_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func TestGroupStoreNew(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	defer os.Remove(db.Path())

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)
}

func TestGroupStorePut(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	err = s.Put(ctx, g)
	a.NoError(err)
}

func TestGroupStoreGetByID(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	err = s.Put(ctx, g)
	a.NoError(err)

	fg, err := s.GetByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ID, fg.ID)
	a.Equal(usermanager.GKGroup, fg.Kind)
	a.Equal("test_group", fg.Key)
	a.Equal("test group", fg.Name)
}

func TestGroupStoreGetAll(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
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

	err = s.Put(ctx, g1)
	a.NoError(err)

	err = s.Put(ctx, g2)
	a.NoError(err)

	err = s.Put(ctx, g3)
	a.NoError(err)

	gs, err := s.GetAll(ctx)
	a.NoError(err)
	a.Len(gs, 3)

	a.Equal(g1.ID, gs[0].ID)
	a.Equal(usermanager.GKGroup, gs[0].Kind)
	a.Equal("test_group", gs[0].Key)
	a.Equal("test group", gs[0].Name)

	a.Equal(g2.ID, gs[1].ID)
	a.Equal(usermanager.GKRole, gs[1].Kind)
	a.Equal("test_role", gs[1].Key)
	a.Equal("test role", gs[1].Name)

	a.Equal(g3.ID, gs[2].ID)
	a.Equal(usermanager.GKGroup, gs[2].Kind)
	a.Equal("test_group123", gs[2].Key)
	a.Equal("test group 123", gs[2].Name)
}

func TestGroupStoreDelete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	err = s.Put(ctx, g)
	a.NoError(err)

	fg, err := s.GetByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ID, fg.ID)
	a.Equal(usermanager.GKGroup, fg.Kind)
	a.Equal("test_group", fg.Key)
	a.Equal("test group", fg.Name)

	err = s.Delete(ctx, g.ID)
	a.NoError(err)

	fg, err = s.GetByID(ctx, g.ID)
	a.Nil(fg)
	a.Error(err)
	a.EqualError(err, usermanager.ErrGroupNotFound.Error())
}

func TestGroupStorePutRelation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	u, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u)

	err = s.PutRelation(ctx, g.ID, u.ID)
	a.NoError(err)
}

func TestGroupStoreHasRelation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	u, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u)

	yes, err := s.HasRelation(ctx, g.ID, u.ID)
	a.False(yes)
	a.EqualError(err, usermanager.ErrRelationNotFound.Error())

	err = s.Put(ctx, g)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u.ID)
	a.NoError(err)

	fg, err := s.GetByID(ctx, g.ID)
	a.NoError(err)
	a.NotNil(fg)
	a.Equal(g.ID, fg.ID)
	a.Equal(usermanager.GKGroup, fg.Kind)
	a.Equal("test_group", fg.Key)
	a.Equal("test group", fg.Name)

	yes, err = s.HasRelation(ctx, g.ID, u.ID)
	a.True(yes)
	a.NoError(err)
}

func TestGroupStoreGetAllRelation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	u1, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	err = s.PutRelation(ctx, g.ID, u1.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u2.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u3.ID)
	a.NoError(err)

	rs, err := s.GetAllRelation(ctx)
	a.NoError(err)
	a.Contains(rs, g.ID)
	a.Equal(rs[g.ID][0], u1.ID)
	a.Equal(rs[g.ID][1], u2.ID)
	a.Equal(rs[g.ID][2], u3.ID)
}

func TestGroupStoreGetRelationByGroupID(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	g2, err := usermanager.NewGroup(usermanager.GKGroup, "test_group_2", "test group 2", nil)
	a.NotNil(g)
	a.NoError(err)

	u1, err := usermanager.NewUser("testuser1", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := usermanager.NewUser("testuser2", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := usermanager.NewUser("testuser3", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	u4, err := usermanager.NewUser("testuser4", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	u5, err := usermanager.NewUser("testuser5", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	err = s.PutRelation(ctx, g.ID, u1.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u2.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u3.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g2.ID, u4.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g2.ID, u5.ID)
	a.NoError(err)

	// group 1
	rs, err := s.GetRelationByGroupID(ctx, g.ID)
	a.NoError(err)
	a.Contains(rs, g.ID)
	a.Equal(rs[g.ID][0], u1.ID)
	a.Equal(rs[g.ID][1], u2.ID)
	a.Equal(rs[g.ID][2], u3.ID)

	// group 2
	rs, err = s.GetRelationByGroupID(ctx, g2.ID)
	a.NoError(err)
	a.Contains(rs, g2.ID)
	a.Equal(rs[g2.ID][0], u4.ID)
	a.Equal(rs[g2.ID][1], u5.ID)
}

func TestGroupStoreDeleteRelation(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	u, err := usermanager.NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u)

	yes, err := s.HasRelation(ctx, g.ID, u.ID)
	a.False(yes)
	a.EqualError(err, usermanager.ErrRelationNotFound.Error())

	err = s.Put(ctx, g)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u.ID)
	a.NoError(err)

	yes, err = s.HasRelation(ctx, g.ID, u.ID)
	a.True(yes)
	a.NoError(err)

	err = s.DeleteRelation(ctx, g.ID, u.ID)
	a.NoError(err)

	yes, err = s.HasRelation(ctx, g.ID, u.ID)
	a.False(yes)
	a.EqualError(err, usermanager.ErrRelationNotFound.Error())
}

func TestGroupStoreDeleteRelationByGroupID(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	ctx := context.Background()

	s, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g, err := usermanager.NewGroup(usermanager.GKGroup, "test_group", "test group", nil)
	a.NotNil(g)
	a.NoError(err)

	g2, err := usermanager.NewGroup(usermanager.GKGroup, "test_group_2", "test group 2", nil)
	a.NotNil(g)
	a.NoError(err)

	u1, err := usermanager.NewUser("testuser1", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := usermanager.NewUser("testuser2", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := usermanager.NewUser("testuser3", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	u4, err := usermanager.NewUser("testuser4", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	u5, err := usermanager.NewUser("testuser5", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u3)

	err = s.PutRelation(ctx, g.ID, u1.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u2.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g.ID, u3.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g2.ID, u4.ID)
	a.NoError(err)

	err = s.PutRelation(ctx, g2.ID, u5.ID)
	a.NoError(err)

	err = s.DeleteRelationByGroupID(ctx, g.ID)
	a.NoError(err)

	// group 1; should be absent now
	rs, err := s.GetRelationByGroupID(ctx, g.ID)
	a.NoError(err)
	a.NotContains(rs, g.ID)

	// group 2
	rs, err = s.GetRelationByGroupID(ctx, g2.ID)
	a.NoError(err)
	a.Contains(rs, g2.ID)
	a.Equal(rs[g2.ID][0], u4.ID)
	a.Equal(rs[g2.ID][1], u5.ID)
}
