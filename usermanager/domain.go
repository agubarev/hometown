package usermanager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/blevesearch/bleve"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
	"github.com/spf13/viper"
	"gitlab.com/agubarev/hometown/util"
)

// permission mode used for the domain storage
var localStorageFileMode = os.FileMode(0755)

// DomainOptions is a configuration object for a domain
type DomainOptions struct {
	logger *zap.Logger
}

// DefaultDomainOptions returns domain config with pre-defined default settings
func DefaultDomainOptions() (*DomainOptions, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	dc := &DomainOptions{
		logger: logger,
	}

	return dc, nil
}

// Domain represents a single organizational entity which incorporates
// organizations, users, roles, groups and teams
type Domain struct {
	ID           ulid.ULID       `json:"id"`
	Name         string          `json:"n"`
	Owner        *User           `json:"-"`
	Users        *UserContainer  `json:"-"`
	Groups       *GroupContainer `json:"-"`
	AccessPolicy *AccessPolicy   `json:"-"`

	CreatedAt  time.Time `json:"t_cr"`
	ModifiedAt time.Time `json:"t_up,omitempty"`

	config *viper.Viper
	logger *zap.Logger
	sync.RWMutex
}

func localStorageDirectory() string {
	return viper.GetString("instance.domains.directory")
}

func validateLocalStorageDirectory() error {
	// domain storage directory must be set
	lsd := localStorageDirectory()
	if lsd == "" {
		return fmt.Errorf("local domain storage directory is not defined")
	}

	// domain storage directory must exist during validation
	if err := util.CreateDirectoryIfNotExists(lsd, localStorageFileMode); err != nil {
		return fmt.Errorf("failed to create local domain storage directory: %s", err)
	}

	// domain storage must be readable by the instance
	if _, err := ioutil.ReadDir(lsd); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("local storage directory is not readable")
		}

		return err
	}

	return nil
}

func initLocalDatabase(dbpath string) (*badger.DB, error) {
	// using default options
	opts := badger.DefaultOptions
	opts.Dir = dbpath
	opts.ValueDir = dbpath

	// password storage directory must exist and be writable
	if err := util.CreateDirectoryIfNotExists(dbpath, localStorageFileMode); err != nil {
		return nil, fmt.Errorf("failed to create local password database directory: %s", err)
	}

	// attempting to open password database
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open local passwords database: %s", err)
	}

	return db, nil
}

func initLocalIndex(indexDir string) (bleve.Index, error) {
	var index bleve.Index

	// bleve requires a directory instead of just a file
	indexPath := filepath.Join(indexDir)

	// creating new index; will return an error if the index has already been created
	index, err := bleve.New(indexPath, bleve.NewIndexMapping())
	if err != nil {
		if err == bleve.ErrorIndexPathExists {
			// index already exists, trying to load instead
			index, err = bleve.Open(indexPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open existing index: %s", err)
			}
		} else {
			// unhandled error
			return nil, fmt.Errorf("failed to create new index: %s", err)
		}
	}

	return index, nil
}

// NewDomain initializing new domain
func NewDomain(owner *User, c *DomainOptions) (*Domain, error) {
	// each new domain must have an owner user assigned
	if owner == nil {
		return nil, fmt.Errorf("new domain must have an owner: %s", ErrNilUser)
	}

	// if config is nil then using default settings
	if c == nil {
		dc, err := DefaultDomainOptions()
		if err != nil {
			return nil, fmt.Errorf("failed to obtain default domain configuration: %s", err)
		}
		c = dc
	}

	// new domain
	d := &Domain{
		ID: util.NewULID(),

		// initial domain owner, full access to this domain
		Owner: owner,

		// at this moment each policy is independent by default
		// it doesn't have a parent by default, doesn't inherit nor extends anything
		// TODO think through the default initial state of an AccessPolicy
		// TODO: add domain ID as policy ID
		AccessPolicy: NewAccessPolicy(owner, nil, false, false),

		// domain-level logger
		logger: c.logger,
	}

	err := d.init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize new domain: %s", err)
	}

	// creating default groups (basically, role groups)
	err = d.setupDefaultGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to setup default groups")
	}

	// configuring default groups
	superuserRole, err := d.Groups.GetByKey("superuser")
	if err != nil {
		return nil, fmt.Errorf("failed to obtain superuser role: %s", err)
	}

	// TODO: store access policy object

	// giving full access rights to the superuser role
	d.AccessPolicy.SetRoleRights(owner, superuserRole, APFullAccess)

	// creating initial access policies
	// TODO: set owner rights
	// TODO: set super role rights

	return d, nil
}

// LoadDomain loads existing domain
func LoadDomain(id ulid.ULID) (*Domain, error) {
	// setting just the ID which is sufficient to start
	// the initialization mechanism
	d := &Domain{
		ID: id,
	}

	if err := d.init(); err != nil {
		return nil, fmt.Errorf("failed to load domain (%s): %s", id, err)
	}

	// TODO: load domain config
	// TODO: obtain owner
	// TODO: obtain metadata

	return d, nil
}

// Logger returns domain logger
func (d *Domain) Logger() *zap.Logger {
	return d.logger
}

func (d *Domain) initConfig() error {
	// only doing it once
	if d.config != nil {
		// config is already initialized, aborting
		return nil
	}

	// safety first checks
	if d == nil {
		return ErrNilDomain
	}

	if d.Owner == nil {
		return ErrNilUser
	}

	configFilepath := filepath.Join(d.StoragePath(), fmt.Sprintf("%s_config.yaml", d.StringID()))
	if _, err := os.Stat(configFilepath); err != nil {
		if os.IsNotExist(err) {
			// just creating the empty config file
			d.logger.Info("creating domain config", zap.String("path", configFilepath))
			if err := ioutil.WriteFile(configFilepath, []byte(""), localStorageFileMode); err != nil {
				return fmt.Errorf("failed to create domain config: %s", err)
			}
		}
	}

	// new config instance
	c := viper.New()
	c.SetConfigName(fmt.Sprintf("%s_config", d.StringID()))
	c.AddConfigPath(d.StoragePath())
	c.SetConfigType("yaml")

	// setting as domain config
	d.config = c

	return nil
}

func (d *Domain) saveConfig() error {
	if err := d.initConfig(); err != nil {
		return fmt.Errorf("failed to initialize domain config: %s", err)
	}

	// domain
	d.config.Set("domain.id", d.StringID())
	d.config.Set("domain.name", d.Name)

	// owner
	d.config.Set("owner.id", d.Owner.ID)
	d.config.Set("owner.username", d.Owner.Username)
	d.config.Set("owner.email", d.Owner.Email)

	// metadata
	d.config.Set("times.created_at", d.CreatedAt)
	d.config.Set("times.modified_at", d.ModifiedAt)

	// saving domain config
	if err := d.config.WriteConfig(); err != nil {
		return fmt.Errorf("failed to save config: %s", err)
	}

	return nil
}

func (d *Domain) loadConfig() error {
	if err := d.initConfig(); err != nil {
		return fmt.Errorf("failed to initialize domain config: %s", err)
	}

	// loading domain config
	d.logger.Info("reading domain config", zap.String("config", d.config.ConfigFileUsed()))
	if err := d.config.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to load config: %s", err)
	}

	return nil
}

// Init initializes domain
func (d *Domain) init() error {
	// deferring log flush
	defer d.logger.Sync()

	d.logger.Info("initializing domain")

	//---------------------------------------------------------------------------
	// initializing local storage
	//---------------------------------------------------------------------------
	// precautious check
	d.logger.Info("validating local storage directory")
	if err := validateLocalStorageDirectory(); err != nil {
		return err
	}

	// creating storage if it hasn't been created yet
	sdir := d.StoragePath()
	if err := util.CreateDirectoryIfNotExists(sdir, localStorageFileMode); err != nil {
		return fmt.Errorf("failed to create domain storage: %s", err)
	}

	//---------------------------------------------------------------------------
	// initializing local databases
	//---------------------------------------------------------------------------
	d.logger.Info("initializing storage")

	// general database
	db, err := initLocalDatabase(filepath.Join(d.StoragePath(), "data"))
	if err != nil {
		return err
	}

	// password database
	pdb, err := initLocalDatabase(filepath.Join(d.StoragePath(), "passwords"))
	if err != nil {
		return err
	}

	//---------------------------------------------------------------------------
	// initializing stores and indexes
	//---------------------------------------------------------------------------
	d.logger.Info("initializing domain stores")

	us, err := NewDefaultUserStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %s", err)
	}

	gs, err := NewDefaultGroupStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize group store: %s", err)
	}

	uidx, err := initLocalIndex(filepath.Join(d.StoragePath(), "index", "users"))
	if err != nil {
		return err
	}

	/*
		aps, err := NewDefaultAccessPolicyStore(db)
		if err != nil {
			return fmt.Errorf("failed to initialize access policy store: %s", err)
		}
	*/

	ps, err := NewDefaultPasswordStore(pdb)
	if err != nil {
		return fmt.Errorf("failed to initialize password store: %s", err)
	}

	//---------------------------------------------------------------------------
	// initializing and validating containers
	//---------------------------------------------------------------------------

	d.logger.Info("initializing user container")
	uc, err := NewUserContainer(us, uidx)
	if err != nil {
		return fmt.Errorf("failed to initialize user container: %s", err)
	}

	// initializing and assigning a password manager to the user container
	pm, err := NewDefaultPasswordManager(ps)
	if err != nil {
		return fmt.Errorf("failed to initialize user password manager: %s", err)
	}

	if err = uc.SetPasswordManager(pm); err != nil {
		return err
	}

	if err = uc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate user container: %s", err)
	}

	d.logger.Info("initializing group container")
	gc, err := NewGroupContainer(gs)
	if err != nil {
		return fmt.Errorf("failed to initialize group container: %s", err)
	}

	if err = gc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate group container: %s", err)
	}

	if err = d.saveConfig(); err != nil {
		return fmt.Errorf("failed to initialize domain configuration: %s", err)
	}

	// TODO: set owner properly
	// TODO: load groups
	// TODO: load users

	//---------------------------------------------------------------------------
	// finalizing domain initialization
	//---------------------------------------------------------------------------

	if err = d.SetUserContainer(uc); err != nil {
		return err
	}

	if err = d.SetGroupContainer(gc); err != nil {
		return err
	}

	return nil
}

func (d *Domain) setupDefaultGroups() error {
	if d.Groups == nil {
		return ErrNilGroupContainer
	}

	// regular user
	userRole, err := NewGroup(GKRole, "user", "Regular User", nil)
	if err != nil {
		return fmt.Errorf("failed to create regular user role: %s", err)
	}

	err = d.Groups.Add(userRole)
	if err != nil {
		return err
	}

	// manager
	managerRole, err := NewGroup(GKRole, "manager", "Manager", userRole)
	if err != nil {
		return fmt.Errorf("failed to create manager role: %s", err)
	}

	err = d.Groups.Add(managerRole)
	if err != nil {
		return err
	}

	// superuser
	superuserRole, err := NewGroup(GKRole, "superuser", "Super User", managerRole)
	if err != nil {
		return fmt.Errorf("failed to create superuser role: %s", err)
	}

	err = d.Groups.Add(superuserRole)
	if err != nil {
		return err
	}

	return nil
}

// StringID is a short text info
func (d *Domain) StringID() string {
	return fmt.Sprintf("%s", d.ID)
}

// SetUserContainer assigns a user container to this domain
func (d *Domain) SetUserContainer(uc *UserContainer) error {
	if uc == nil {
		return ErrNilUserContainer
	}

	err := uc.Validate()
	if err != nil {
		return fmt.Errorf("failed to set user container: %s", err)
	}

	err = uc.SetDomain(d)
	if err != nil {
		return fmt.Errorf("failed to set user container: %s", err)
	}

	// assigning user container to this domain
	d.Users = uc

	return nil
}

// SetGroupContainer assigns a user container to this domain
func (d *Domain) SetGroupContainer(gc *GroupContainer) error {
	if gc == nil {
		return ErrNilGroupContainer
	}

	err := gc.Validate()
	if err != nil {
		return fmt.Errorf("failed to set group container: %s", err)
	}

	err = gc.SetDomain(d)
	if err != nil {
		return fmt.Errorf("failed to set group container: %s", err)
	}

	d.Groups = gc

	return nil
}

// StorageID returns a string which should be used to identify
// this domain inside a respective storage, i.e. directory name
// inside a local domain storage directory
func (d *Domain) StorageID() string {
	return fmt.Sprintf("%s", d.ID)
}

// StoragePath returns domain's local storage directory path
func (d *Domain) StoragePath() string {
	return filepath.Join(localStorageDirectory(), d.StorageID())
}
