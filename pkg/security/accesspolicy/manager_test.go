package accesspolicy_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicyManager(t *testing.T) {
	a := assert.New(t)

	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
	// creating a normal policy with type name and ID set, no key
	//---------------------------------------------------------------------------
	key := accesspolicy.TKey{}
	typeName := accesspolicy.NewObjectName("with type and id, no key")
	objectID := uint32(1)

	ap, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// policy without an owner
	//---------------------------------------------------------------------------
	key = accesspolicy.TKey{}
	typeName = accesspolicy.NewObjectName("policy without an owner")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		0,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating a policy with a key, object type and ID set
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("test key")
	typeName = accesspolicy.NewObjectName("with type, id and key")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	//---------------------------------------------------------------------------
	// creating the same policy with the same object type and id
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("test key (to attempt duplication)")
	typeName = accesspolicy.NewObjectName("with type and id (to attempt duplication)")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// attempting to create a policy with a duplicate key
	// NOTE: must fail
	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyKeyTaken, err.Error())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy with an object name without object ID
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("with name but without id")
	typeName = accesspolicy.NewObjectName("test object")
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy without an object name but with object ID set
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("without name but with id")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)

	//---------------------------------------------------------------------------
	// creating a re-usable parent policy
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("re-usable parent policy")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	basePolicy, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(basePolicy)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(typeName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	//---------------------------------------------------------------------------
	// attempting to create a proper policy but with a non-existing parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy with non-existing parent")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		123321,   // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// inheritance without a parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy inherits with no parent")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                   // key
		1,                     // owner ID
		123321,                // parent ID
		objectID,              // object ID
		typeName,              // object type
		accesspolicy.FInherit, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// extension without a parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy extends with no parent")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                  // key
		1,                    // owner ID
		123321,               // parent ID
		objectID,             // object ID
		typeName,             // object type
		accesspolicy.FExtend, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper but inherits and extends at the same time
	// NOTE: must fail
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy inherits and extends (must not be created)")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,           // key
		1,             // owner ID
		basePolicy.ID, // parent ID
		objectID,      // object ID
		typeName,      // object type
		accesspolicy.FInherit|accesspolicy.FExtend, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper creation with inheritance only
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("proper policy with inheritance")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                   // key
		1,                     // owner ID
		basePolicy.ID,         // parent ID
		objectID,              // object ID
		typeName,              // object type
		accesspolicy.FInherit, // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.NotZero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.True(ap.IsInherited())
	a.False(ap.IsExtended())

	//---------------------------------------------------------------------------
	// proper creation with extension only
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("proper policy with extension")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                  // key
		1,                    // owner ID
		basePolicy.ID,        // parent ID
		objectID,             // object ID
		typeName,             // object type
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.NotZero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.True(ap.IsExtended())
}

func TestAccessPolicyManagerUpdate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
	// test policy
	//---------------------------------------------------------------------------
	key := accesspolicy.NewKey("test policy")
	typeName := accesspolicy.TObjectName{}
	objectID := uint32(0)

	ap, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating base policy (to be used as a parent)
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("base policy")
	typeName = accesspolicy.TObjectName{}
	objectID = uint32(0)

	basePolicy, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(basePolicy.ID)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(typeName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	// setting parent
	a.NoError(m.SetParent(ctx, ap.ID, basePolicy.ID))

	// re-obtaining updated policy
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, ap.ParentID)

	// re-obtaining updated policy
	// NOTE: parent must be set
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, ap.ParentID)
	a.Equal(ap.ParentID, basePolicy.ID)

	// unsetting parent
	a.NoError(m.SetParent(ctx, ap.ID, 0))

	// re-obtaining updated policy
	// NOTE: parent must be cleared
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(uint32(0), ap.ParentID)
	a.Zero(ap.ParentID)

	// key, object name and id must not be changeable
	a.Error(ap.SetKey(accesspolicy.NewKey("new key"), 32))
	a.Error(ap.SetObjectName(accesspolicy.NewObjectName("new object name"), 32))

	// attempting to change object id and save
	ap.ObjectName = accesspolicy.NewObjectName("doesn't matter")
	ap.ObjectID = ap.ObjectID + 1
	a.EqualError(accesspolicy.ErrForbiddenChange, m.Update(ctx, ap).Error())

	// re-obtaining policy
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)

	// set assignor rights
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 2, accesspolicy.APView))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APMove))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APCreate))
}

func TestAccessPolicyManagerSetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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

	g1, err := gm.Create(ctx, group.FGroup, 0, "test group 1", "test group 1")
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, 0, "test group 2", "test group 2")
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, 0, "test role 1", "test role 1")
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, 0, "test role 2", "test role 2")
	a.NoError(err)

	// expected rights
	wantedRights := accesspolicy.APCreate | accesspolicy.APView | accesspolicy.APDelete

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.NewKey("test policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectName{},         // object type
		0,                                  // flags
	)
	a.NoError(err)

	// public
	a.NoError(m.SetRights(ctx, accesspolicy.SKEveryone, ap.ID, 1, 2, wantedRights))

	// users
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 2, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 3, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 4, wantedRights))

	// roles
	a.NoError(m.SetRights(ctx, accesspolicy.SKRoleGroup, ap.ID, 1, r1.ID, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKRoleGroup, ap.ID, 1, r2.ID, wantedRights))

	// groups
	a.NoError(m.SetRights(ctx, accesspolicy.SKGroup, ap.ID, 1, g1.ID, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKGroup, ap.ID, 1, g2.ID, wantedRights))

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
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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

	g1, err := gm.Create(ctx, group.FGroup, 0, "test group 1", "test group 1")
	a.NoError(err)

	g2, err := gm.Create(ctx, group.FGroup, 0, "test group 2", "test group 2")
	a.NoError(err)

	r1, err := gm.Create(ctx, group.FRole, 0, "test role 1", "test role 1")
	a.NoError(err)

	r2, err := gm.Create(ctx, group.FRole, 0, "test role 2", "test role 2")
	a.NoError(err)

	// expected rights
	wantedRights := accesspolicy.APView | accesspolicy.APChange | accesspolicy.APDelete | accesspolicy.APCopy

	//---------------------------------------------------------------------------
	// creating policy and setting rights
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.NewKey("test policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectName{},         // object type
		0,                                  // flags
	)
	a.NoError(err)

	// public
	a.NoError(m.SetRights(ctx, accesspolicy.SKEveryone, ap.ID, 1, 2, wantedRights))

	// users
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 2, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 3, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 4, wantedRights))

	// roles
	a.NoError(m.SetRights(ctx, accesspolicy.SKRoleGroup, ap.ID, 1, r1.ID, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKRoleGroup, ap.ID, 1, r2.ID, wantedRights))

	// groups
	a.NoError(m.SetRights(ctx, accesspolicy.SKGroup, ap.ID, 1, g1.ID, wantedRights))
	a.NoError(m.SetRights(ctx, accesspolicy.SKGroup, ap.ID, 1, g2.ID, wantedRights))

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

	fetchedPolicy, err = m.PolicyByObject(ctx, ap.ObjectName, ap.ObjectID)
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
	a.EqualError(accesspolicy.ErrPolicyNotFound, errors.Cause(err).Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByKey(ctx, ap.Key)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = m.PolicyByObject(ctx, ap.ObjectName, ap.ObjectID)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)
}
