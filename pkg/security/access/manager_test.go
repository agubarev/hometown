package access_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/access"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicyManager(t *testing.T) {
	a := assert.New(t)

	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)
}

func TestAccessPolicyManagerCreate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// proceeding with the test
	// creating a normal policy with type name and ActorID set, no key
	//---------------------------------------------------------------------------
	key := access.TKey{}
	objectName := access.ObjectName("with type and id, no key")
	objectID := uuid.New()

	p, err := m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotZero(p.ID)
	a.Zero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(access.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// policy without an owner
	//---------------------------------------------------------------------------
	key = access.TKey{}
	objectName = access.ObjectName("policy without an owner")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.Nil,                               // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotZero(p.ID)
	a.Zero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(access.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating a policy with a key,  object name and ActorID set
	//---------------------------------------------------------------------------
	key = access.Key("test key")
	objectName = access.ObjectName("with type, id and key")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Zero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	//---------------------------------------------------------------------------
	// creating the same policy with the same  object name and id
	//---------------------------------------------------------------------------
	key = access.Key("test key (to attempt duplication)")
	objectName = access.ObjectName("with type and id (to attempt duplication)")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Zero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// attempting to create a policy with a duplicate key
	// NOTE: must fail
	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.Error(err)
	a.EqualError(access.ErrPolicyKeyTaken, err.Error())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy with an object name without object
	//---------------------------------------------------------------------------
	key = access.Key("with name but without id")
	objectName = access.ObjectName("test object")
	objectID = uuid.Nil

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy without an object name but with  objectset
	//---------------------------------------------------------------------------
	key = access.Key("without name but with id")
	objectName = access.TObjectName{}
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)

	//---------------------------------------------------------------------------
	// creating a re-usable parent policy
	//---------------------------------------------------------------------------
	key = access.Key("re-usable parent policy")
	objectName = access.TObjectName{}
	objectID = uuid.Nil

	basePolicy, err := m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotNil(basePolicy)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(objectName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	//---------------------------------------------------------------------------
	// attempting to create a proper policy but with a non-existing parent
	//---------------------------------------------------------------------------
	key = access.Key("policy with non-existing parent")
	objectName = access.TObjectName{}
	objectID = uuid.Nil

	p, err = m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.New(),                             // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// inheritance without a parent
	//---------------------------------------------------------------------------
	key = access.Key("policy inherits with no parent")
	objectName = access.TObjectName{}
	objectID = uuid.Nil
	ownerID := uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		ownerID,                                // owner
		uuid.New(),                             // parent
		access.NewObject(objectID, objectName), // object
		access.FInherit,                        // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// extension without a parent
	//---------------------------------------------------------------------------
	key = access.Key("policy extends with no parent")
	objectName = access.TObjectName{}
	objectID = uuid.Nil
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		ownerID,                                // owner
		uuid.New(),                             // parent
		access.NewObject(objectID, objectName), // object
		access.FExtend,                         // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper but inherits and extends at the same time
	// NOTE: must fail
	//---------------------------------------------------------------------------
	key = access.Key("policy inherits and extends (must not be created)")
	objectName = access.TObjectName{}
	objectID = uuid.Nil
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		ownerID,                                // owner
		basePolicy.ID,                          // parent
		access.NewObject(objectID, objectName), // object
		access.FInherit|access.FExtend,         // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper creation with inheritance only
	//---------------------------------------------------------------------------
	key = access.Key("proper policy with inheritance")
	objectName = access.TObjectName{}
	objectID = uuid.Nil
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		ownerID,                                // owner
		basePolicy.ID,                          // parent
		access.NewObject(objectID, objectName), // object
		access.FInherit,                        // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.NotZero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.True(p.IsInherited())
	a.False(p.IsExtended())

	//---------------------------------------------------------------------------
	// proper creation with extension only
	//---------------------------------------------------------------------------
	key = access.Key("proper policy with extension")
	objectName = access.TObjectName{}
	objectID = uuid.Nil
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                                    // key
		ownerID,                                // owner
		basePolicy.ID,                          // parent
		access.NewObject(objectID, objectName), // object
		access.FExtend,                         // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.NotZero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.True(p.IsExtended())
}

func TestAccessPolicyManagerUpdate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	key := access.Key("test policy")
	objectName := access.TObjectName{}
	objectID := uuid.Nil

	p, err := m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotZero(p.ID)
	a.Zero(p.ParentID)
	a.Equal(key, p.Key)
	a.Equal(objectName, p.ObjectName)
	a.Equal(objectID, p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(access.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating base policy (to be used as a parent)
	//---------------------------------------------------------------------------
	key = access.Key("base policy")
	objectName = access.TObjectName{}
	objectID = uuid.Nil

	basePolicy, err := m.Create(
		ctx,
		key,                                    // key
		uuid.New(),                             // owner
		uuid.Nil,                               // parent
		access.NewObject(objectID, objectName), // object
		0,                                      // flags
	)
	a.NoError(err)
	a.NotZero(basePolicy.ID)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(objectName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	// setting parent
	a.NoError(m.SetParent(ctx, p.ID, basePolicy.ID))

	// re-obtaining updated policy
	p, err = m.PolicyByID(ctx, p.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, p.ParentID)

	// re-obtaining updated policy
	// NOTE: parent must be set
	p, err = m.PolicyByID(ctx, p.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, p.ParentID)
	a.Equal(p.ParentID, basePolicy.ID)

	// unsetting parent
	a.NoError(m.SetParent(ctx, p.ID, uuid.Nil))

	// re-obtaining updated policy
	// NOTE: parent must be cleared
	p, err = m.PolicyByID(ctx, p.ID)
	a.NoError(err)
	a.Equal(uuid.Nil, p.ParentID)
	a.Zero(p.ParentID)

	// key, object name and id must not be changeable
	a.Error(p.SetKey(access.Key("new key"), 32))
	a.Error(p.SetObjectName(access.ObjectName("new object name"), 32))

	// attempting to rosterChange object id and save
	p.ObjectName = access.ObjectName("doesn't matter")
	p.ObjectID = uuid.New()
	a.EqualError(access.ErrForbiddenChange, m.Update(ctx, p).Error())

	// re-obtaining policy
	p, err = m.PolicyByID(ctx, p.ID)
	a.NoError(err)

	act1 := access.NewActor(access.AUser, uuid.New())
	act2 := access.NewActor(access.AUser, uuid.New())

	// set assignor rights
	a.NoError(m.GrantAccess(ctx, p.ID, act1, act2, access.APView))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, act2, access.APView))
	a.False(m.HasRights(ctx, p.ID, act2, access.APChange))
	a.False(m.HasRights(ctx, p.ID, act2, access.APMove))
	a.False(m.HasRights(ctx, p.ID, act2, access.APDelete))
	a.False(m.HasRights(ctx, p.ID, act2, access.APCreate))
}

func TestAccessPolicyManagerSetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test group 1"), group.Name("test group 1"))
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test group 2"), group.Name("test group 2"))
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test role 1"), group.Name("test role 1"))
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test role 2"), group.Name("test role 2"))
	a.NoError(err)

	// expected rights
	wantedRights := access.APCreate | access.APView | access.APDelete

	act1 := access.NewActor(access.AUser, uuid.New())
	act2 := access.NewActor(access.AUser, uuid.New())
	act3 := access.NewActor(access.AUser, uuid.New())
	act4 := access.NewActor(access.AUser, uuid.New())

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		act1.ID,                   // owner
		uuid.Nil,                  // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)

	// public
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act2, wantedRights))

	// users
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act2, wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act3, wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act4, wantedRights))

	// roles
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, r1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, r2.ID), wantedRights))

	// groups
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, g1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, g2.ID), wantedRights))

	// persisting changes
	a.NoError(m.Update(ctx, ap))
}

func TestAccessPolicyManagerDelete(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test group 1"), group.Name("test group 1"))
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test group 2"), group.Name("test group 2"))
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test role 1"), group.Name("test role 1"))
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test role 2"), group.Name("test role 2"))
	a.NoError(err)

	// expected rights
	wantedRights := access.APView | access.APChange | access.APDelete | access.APCopy

	ownerID := uuid.New()

	act1 := access.NewActor(access.AUser, uuid.New())
	act2 := access.NewActor(access.AUser, uuid.New())
	act3 := access.NewActor(access.AUser, uuid.New())
	act4 := access.NewActor(access.AUser, uuid.New())

	//---------------------------------------------------------------------------
	// creating policy and setting rights
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		ownerID,                   // owner
		uuid.Nil,                  // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)

	// public
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act2, wantedRights))

	// users
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act2, wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act3, wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, act4, wantedRights))

	// roles
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, r1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, r2.ID), wantedRights))

	// groups
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, g1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, access.NewActor(access.ARoleGroup, g2.ID), wantedRights))

	// persisting changes
	a.NoError(m.Update(ctx, ap))

	//---------------------------------------------------------------------------
	// making sure it's inside the container
	//---------------------------------------------------------------------------
	fetchedPolicy, err := m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	fetchedPolicy, err = m.PolicyByKey(ctx, ap.Key)
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	fetchedPolicy, err = m.PolicyByObject(ctx, access.NewObject(ap.ObjectID, ap.ObjectName))
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	//---------------------------------------------------------------------------
	// deleting policy
	//---------------------------------------------------------------------------
	a.NoError(m.DeletePolicy(ctx, ap))

	//---------------------------------------------------------------------------
	// attempting to get policies after their deletion
	//---------------------------------------------------------------------------
	fetchedPolicy, err = m.PolicyByID(ctx, ap.ID)
	a.Error(err)
	a.EqualError(access.ErrPolicyNotFound, errors.Cause(err).Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByKey(ctx, ap.Key)
	a.Error(err)
	a.EqualError(access.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByObject(ctx, access.NewObject(ap.ObjectID, ap.ObjectName))
	a.Error(err)
	a.EqualError(access.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)
}
