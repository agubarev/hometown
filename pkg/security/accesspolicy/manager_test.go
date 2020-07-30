package accesspolicy_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/agubarev/hometown/pkg/util/bytearray"
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// proceeding with the test
	// creating a normal policy with type name and ActorID set, no key
	//---------------------------------------------------------------------------
	key := bytearray.NilByteString32
	objectName := bytearray.NewByteString32("with type and id, no key")
	objectID := uuid.New()

	p, err := m.Create(
		ctx,
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// policy without an owner
	//---------------------------------------------------------------------------
	key = bytearray.NilByteString32
	objectName = bytearray.NewByteString32("policy without an owner")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,      // key
		uuid.Nil, // owner
		uuid.Nil, // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating a policy with a key,  object name and ActorID set
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("test key")
	objectName = bytearray.NewByteString32("with type, id and key")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
	key = bytearray.NewByteString32("test key (to attempt duplication)")
	objectName = bytearray.NewByteString32("with type and id (to attempt duplication)")
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
	)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyKeyTaken, err.Error())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy with an object name without object
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("with name but without id")
	objectName = bytearray.NewByteString32("test object")
	objectID = uuid.Nil

	p, err = m.Create(
		ctx,
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy without an object name but with object ID set
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("without name but with id")
	objectName = bytearray.NilByteString32
	objectID = uuid.New()

	p, err = m.Create(
		ctx,
		key,        // key
		uuid.New(), // owner
		uuid.Nil,   // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, p.ID)
	a.Error(err)

	//---------------------------------------------------------------------------
	// creating a re-usable parent policy
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("re-usable parent policy")

	basePolicy, err := m.Create(
		ctx,
		key,                      // key
		uuid.New(),               // owner
		uuid.Nil,                 // parent
		accesspolicy.NilObject(), // object
		0,                        // flags
	)
	a.NoError(err)
	a.NotNil(basePolicy)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	//---------------------------------------------------------------------------
	// attempting to create a proper policy but with a non-existing parent
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("policy with non-existing parent")

	p, err = m.Create(
		ctx,
		key,                      // key
		uuid.New(),               // owner
		uuid.New(),               // parent
		accesspolicy.NilObject(), // object
		0,                        // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// inheritance without a parent
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("policy inherits with no parent")
	objectName = bytearray.NewByteString32("test object name 3")
	objectID = uuid.Nil
	ownerID := uuid.New()

	p, err = m.Create(
		ctx,
		key,                      // key
		ownerID,                  // owner
		uuid.New(),               // parent
		accesspolicy.NilObject(), // object
		accesspolicy.FInherit,    // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// extension without a parent
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("policy extends with no parent")
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                      // key
		ownerID,                  // owner
		uuid.New(),               // parent
		accesspolicy.NilObject(), // object
		accesspolicy.FExtend,     // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper but inherits and extends at the same time
	// NOTE: must fail
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("policy inherits and extends (must not be created)")
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                      // key
		ownerID,                  // owner
		basePolicy.ID,            // parent
		accesspolicy.NilObject(), // object
		accesspolicy.FInherit|accesspolicy.FExtend, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper creation with inheritance only
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("proper policy with inheritance")
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                      // key
		ownerID,                  // owner
		basePolicy.ID,            // parent
		accesspolicy.NilObject(), // object
		accesspolicy.FInherit,    // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.NotZero(p.ParentID)
	a.Equal(key, p.Key)
	a.True(p.IsInherited())
	a.False(p.IsExtended())

	//---------------------------------------------------------------------------
	// proper creation with extension only
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("proper policy with extension")
	ownerID = uuid.New()

	p, err = m.Create(
		ctx,
		key,                      // key
		ownerID,                  // owner
		basePolicy.ID,            // parent
		accesspolicy.NilObject(), // object
		accesspolicy.FExtend,     // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.NotZero(p.ParentID)
	a.Equal(key, p.Key)
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	key := bytearray.NewByteString32("test policy")
	objectName := bytearray.NilByteString32
	objectID := uuid.Nil

	p, err := m.Create(
		ctx,
		key,      // key
		act1.ID,  // owner
		uuid.Nil, // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating base policy (to be used as a parent)
	//---------------------------------------------------------------------------
	key = bytearray.NewByteString32("base policy")
	objectName = bytearray.NilByteString32
	objectID = uuid.Nil

	basePolicy, err := m.Create(
		ctx,
		key,      // key
		act1.ID,  // owner
		uuid.Nil, // parent
		accesspolicy.NewObject(objectID, objectName), // object
		0, // flags
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
	a.Error(p.SetKey(bytearray.NewByteString32("new key")))
	a.Error(p.SetObjectName(bytearray.NewByteString32("new object name")))

	// attempting to rosterChange object id and save
	p.ObjectName = bytearray.NewByteString32("doesn't matter")
	p.ObjectID = uuid.New()
	a.EqualError(accesspolicy.ErrForbiddenChange, m.Update(ctx, p).Error())

	// re-obtaining policy
	p, err = m.PolicyByID(ctx, p.ID)
	a.NoError(err)

	// set assignor rights
	a.NoError(m.GrantAccess(ctx, p.ID, act1, act2, accesspolicy.APView))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, act2, accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APMove))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APCreate))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, bytearray.NewByteString32("test group 1"), bytearray.NewByteString128("test group 1"))
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, uuid.Nil, bytearray.NewByteString32("test group 2"), bytearray.NewByteString128("test group 2"))
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, uuid.Nil, bytearray.NewByteString32("test role 1"), bytearray.NewByteString128("test role 1"))
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, uuid.Nil, bytearray.NewByteString32("test role 2"), bytearray.NewByteString128("test role 2"))
	a.NoError(err)

	// expected rights
	wantedRights := accesspolicy.APCreate | accesspolicy.APView | accesspolicy.APDelete

	act1 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act2 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act3 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act4 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		bytearray.NewByteString32("test policy"), // key
		act1.ID,                                  // owner
		uuid.Nil,                                 // parent
		accesspolicy.NewObject(uuid.Nil, bytearray.NilByteString32),
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
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, r1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, r2.ID), wantedRights))

	// groups
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, g1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, ap.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, g2.ID), wantedRights))

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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, bytearray.NewByteString32("test group 1"), bytearray.NewByteString128("test group 1"))
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, uuid.Nil, bytearray.NewByteString32("test group 2"), bytearray.NewByteString128("test group 2"))
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, uuid.Nil, bytearray.NewByteString32("test role 1"), bytearray.NewByteString128("test role 1"))
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, uuid.Nil, bytearray.NewByteString32("test role 2"), bytearray.NewByteString128("test role 2"))
	a.NoError(err)

	// expected rights
	wantedRights := accesspolicy.APView | accesspolicy.APChange | accesspolicy.APDelete | accesspolicy.APCopy

	act1 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act2 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act3 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	act4 := accesspolicy.NewActor(accesspolicy.AUser, uuid.New())
	obj := accesspolicy.NewObject(uuid.New(), bytearray.NewByteString32("test object name"))

	//---------------------------------------------------------------------------
	// creating policy and setting rights
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		bytearray.NewByteString32("test policy"), // key
		act1.ID,                                  // owner
		uuid.Nil,                                 // parent
		obj,
		0, // flags
	)
	a.NoError(err)

	// public
	a.NoError(m.GrantAccess(ctx, p.ID, act1, accesspolicy.PublicActor(), wantedRights))

	// users
	a.NoError(m.GrantAccess(ctx, p.ID, act1, act2, wantedRights))
	a.NoError(m.GrantAccess(ctx, p.ID, act1, act3, wantedRights))
	a.NoError(m.GrantAccess(ctx, p.ID, act1, act4, wantedRights))

	// roles
	a.NoError(m.GrantAccess(ctx, p.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, r1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, p.ID, act1, accesspolicy.NewActor(accesspolicy.ARoleGroup, r2.ID), wantedRights))

	// groups
	a.NoError(m.GrantAccess(ctx, p.ID, act1, accesspolicy.NewActor(accesspolicy.AGroup, g1.ID), wantedRights))
	a.NoError(m.GrantAccess(ctx, p.ID, act1, accesspolicy.NewActor(accesspolicy.AGroup, g2.ID), wantedRights))

	// persisting changes
	a.NoError(m.Update(ctx, p))

	//---------------------------------------------------------------------------
	// making sure it's inside the container
	//---------------------------------------------------------------------------
	fetchedPolicy, err := m.PolicyByID(ctx, p.ID)
	a.NoError(err)
	a.True(reflect.DeepEqual(p, fetchedPolicy))

	fetchedPolicy, err = m.PolicyByKey(ctx, p.Key)
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(p, fetchedPolicy))

	fetchedPolicy, err = m.PolicyByObject(ctx, accesspolicy.NewObject(p.ObjectID, p.ObjectName))
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(p, fetchedPolicy))

	//---------------------------------------------------------------------------
	// deleting policy
	//---------------------------------------------------------------------------
	a.NoError(m.DeletePolicy(ctx, p))

	//---------------------------------------------------------------------------
	// attempting to get policies after their deletion
	//---------------------------------------------------------------------------
	fetchedPolicy, err = m.PolicyByID(ctx, p.ID)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyNotFound, errors.Cause(err).Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByKey(ctx, p.Key)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByObject(ctx, accesspolicy.NewObject(p.ObjectID, p.ObjectName))
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)
}
