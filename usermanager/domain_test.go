package usermanager_test

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oklog/ulid"

	"gitlab.com/agubarev/hometown/util"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func init() {
	confstring := `
instance:
    domains:
        directory: /tmp/hometown/domains
`

	viper.SetConfigType("yaml")

	if err := viper.ReadConfig(strings.NewReader(confstring)); err != nil {
		panic(err)
	}
}

func TestNewDomain(t *testing.T) {
	a := assert.New(t)

	owner, err := usermanager.NewUser("testuser", "testuser@example.com", map[string]string{
		"firstname": "Andrei",
		"lastname":  "Gubarev",
	})
	a.NoError(err)
	a.NotNil(owner)

	d, err := usermanager.NewDomain(owner, nil)
	a.NoError(err)
	a.NotNil(d)

	// main domain user, domain creator
	a.NotNil(d.Owner)

	// generally checking the existince of storage related paths and files
	a.NotEmpty(d.StoragePath())
	a.True(util.Exists(d.StoragePath()))
	a.True(util.Exists(filepath.Join(d.StoragePath(), "data")))
	a.True(util.Exists(filepath.Join(d.StoragePath(), "index")))
	a.True(util.Exists(filepath.Join(d.StoragePath(), "passwords")))
	a.True(util.Exists(filepath.Join(d.StoragePath(), fmt.Sprintf("%s_config.yaml", d.ID))))

	// checking whether the main containers are set
	a.NotNil(d.Users)
	a.NotNil(d.Groups)
	a.NotNil(d.Tokens)
}

func TestDomainLoadDomainFullTest(t *testing.T) {
	//---------------------------------------------------------------------------
	// this test consists of the following steps:
	// 1. initialize new domain
	// 2. create users and groups (standard and role groups)
	// 3. assign users to groups and roles
	// 4. load domain
	// 5. compare two domain objects (users, groups, etc.)
	//---------------------------------------------------------------------------
	a := assert.New(t)

	owner, err := usermanager.NewUser("agubarev", "agubarev@example.com", map[string]string{
		"firstname": "Андрей",
		"lastname":  "Губарев",
	})
	a.NoError(err)
	a.NotNil(owner)

	// creating new domain
	d, err := usermanager.NewDomain(owner, nil)
	a.NoError(err)
	a.NotNil(d)

	testpass := "asdo12912n9n9kk19daisdbab123njaa1123b"

	// zeroed ID
	nilID, err := ulid.New(0, strings.NewReader(""))
	a.EqualError(io.EOF, err.Error())

	//---------------------------------------------------------------------------
	// creating test users
	//---------------------------------------------------------------------------
	testuser1, err := d.Users.Create("testuser1", "testuser1@example.com", map[string]string{
		"firstname": "Андрей",
		"lastname":  "Губарев",
	})
	a.NoError(err)
	a.NotNil(testuser1)
	a.False(testuser1.HasPassword())

	testuser2, err := d.Users.Create("testuser2", "testuser2@example.com", map[string]string{
		"firstname": "Andrei",
		"lastname":  "Gubarev",
	})
	a.NoError(err)
	a.NotNil(testuser2)
	a.False(testuser2.HasPassword())

	testuser3wp, err := d.Users.CreateWithPassword("testuser3", "testuser3@example.com", testpass, map[string]string{
		"firstname": "Andrejs",
		"lastname":  "Gubarevs",
	})
	a.NoError(err)
	a.NotNil(testuser3wp)
	a.True(testuser3wp.HasPassword())

	// checking number of existing users
	a.Len(d.Users.List(), 4)

	//---------------------------------------------------------------------------
	// creating groups; standalone and hierarchical (groups and role groups)
	//---------------------------------------------------------------------------
	baseGroup, err := d.Groups.Create(usermanager.GKGroup, "basegroup", "Base Group (standalone)", nil)
	a.NoError(err)
	a.NotNil(baseGroup)
	a.Equal(usermanager.GKGroup, baseGroup.Kind)
	a.Equal("basegroup", baseGroup.Key)
	a.Equal(nilID, baseGroup.ParentID)
	a.Nil(baseGroup.Parent())

	nestedGroup1, err := d.Groups.Create(usermanager.GKGroup, "nested group 1", "Nested Group One (inherits Base Group)", baseGroup)
	a.NoError(err)
	a.NotNil(nestedGroup1)
	a.Equal(usermanager.GKGroup, nestedGroup1.Kind)
	a.Equal("nested group 1", nestedGroup1.Key)
	a.Equal(baseGroup.ID, nestedGroup1.ParentID)
	a.NotNil(nestedGroup1.Parent())
	a.Equal(baseGroup.ID, nestedGroup1.Parent().ID)

	nestedGroup2, err := d.Groups.Create(usermanager.GKGroup, "nested group 2", "Nested Group Two (inherits Base Group)", baseGroup)
	a.NoError(err)
	a.NotNil(nestedGroup2)
	a.Equal(usermanager.GKGroup, nestedGroup2.Kind)
	a.Equal("nested group 2", nestedGroup2.Key)
	a.Equal(baseGroup.ID, nestedGroup2.ParentID)
	a.NotNil(nestedGroup2.Parent())
	a.Equal(baseGroup.ID, nestedGroup2.Parent().ID)

	// role groups (basically a copy of the previous declarations)
	baseRole, err := d.Groups.Create(usermanager.GKRole, "baserole", "Base Role (standalone)", nil)
	a.NoError(err)
	a.NotNil(baseRole)
	a.Equal(usermanager.GKRole, baseRole.Kind)
	a.Equal("baserole", baseRole.Key)
	a.Equal(nilID, baseRole.ParentID)
	a.Nil(baseRole.Parent())

	nestedRole1, err := d.Groups.Create(usermanager.GKRole, "nested role 1", "Nested Role One (inherits Base Group)", baseRole)
	a.NoError(err)
	a.NotNil(nestedRole1)
	a.Equal(usermanager.GKRole, nestedRole1.Kind)
	a.Equal("nested role 1", nestedRole1.Key)
	a.Equal(baseRole.ID, nestedRole1.ParentID)
	a.NotNil(nestedRole1.Parent())
	a.Equal(baseRole.ID, nestedRole1.Parent().ID)

	nestedRole2, err := d.Groups.Create(usermanager.GKRole, "nested role 2", "Nested Role Two (inherits Base Group)", baseRole)
	a.NoError(err)
	a.NotNil(nestedRole2)
	a.Equal(usermanager.GKRole, nestedRole2.Kind)
	a.Equal("nested role 2", nestedRole2.Key)
	a.Equal(baseRole.ID, nestedRole2.ParentID)
	a.NotNil(nestedRole2.Parent())
	a.Equal(baseRole.ID, nestedRole2.Parent().ID)

	for _, g := range d.Groups.List(usermanager.GKAll) {
		fmt.Printf("%s: %s(%s)\n", g.ID, g.Name, g.Key)
	}

	// counting total groups
	a.Len(d.Groups.List(usermanager.GKAll), 6)

	//---------------------------------------------------------------------------
	// assigning users to groups and roles
	// WARNING: this is a set of operations that must be done and reviewed with
	// utmost attention and accuracy.
	//---------------------------------------------------------------------------

	// standard groups
	a.NoError(baseGroup.Add(testuser1))
	a.NoError(nestedGroup1.Add(testuser2))
	a.NoError(nestedGroup2.Add(testuser3wp))

	a.True(baseGroup.IsMember(testuser1))
	a.True(nestedGroup1.IsMember(testuser2))
	a.True(nestedGroup2.IsMember(testuser3wp))

	// role groups (same idea as above)
	a.NoError(baseRole.Add(testuser1))
	a.NoError(nestedRole1.Add(testuser2))
	a.NoError(nestedRole2.Add(testuser3wp))

	a.True(baseRole.IsMember(testuser1))
	a.True(nestedRole1.IsMember(testuser2))
	a.True(nestedRole2.IsMember(testuser3wp))

	// TODO: finish this
	// TODO: finish this
	// TODO: finish this
}
