package usermanager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
	"github.com/spf13/viper"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

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
	Passwords    PasswordManager `json:"-"`
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
	_, err := os.Stat(lsd)
	if err != nil {
		if os.IsNotExist(err) {
			// attempting to create storage directory
			if err := os.MkdirAll(lsd, 0755); err != nil {
				return fmt.Errorf("failed to create local domain storage directory: %s", err)
			}
		}

		return fmt.Errorf("failed to initialize local storage directory: %s", err)
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

func initLocalDatabase(dbfile string) (*bbolt.DB, error) {
	if strings.TrimSpace(dbfile) == "" {
		return nil, fmt.Errorf("database file is not specified")
	}

	db, err := bbolt.Open(dbfile, 0755, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %s", err)
	}

	return db, nil
}

func initLocalPasswordDatabase(dbDir string) (*badger.DB, error) {
	// using default options
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir

	// password storage directory must exist and be writable
	_, err := os.Stat(dbDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create local password database directory: %s", err)
		}
	}

	// attempting to open password database
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open local passwords database: %s", err)
	}

	return db, nil
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

	if err := d.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize new domain: %s", err)
	}

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
			if err := ioutil.WriteFile(configFilepath, []byte(""), 0755); err != nil {
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

	d.logger.Info("initializing domain", zap.String("id", d.StringID()))

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
	if _, err := os.Stat(sdir); err != nil {
		if os.IsNotExist(err) {
			// not found, attempting to create domain storage
			d.logger.Info("creating domain storage", zap.String("id", d.StringID()))
			if err := os.MkdirAll(sdir, 0755); err != nil {
				return fmt.Errorf("failed to create domain storage: %s", err)
			}
		} else {
			return fmt.Errorf("failed to stat domain storage: %s", err)
		}
	}

	//---------------------------------------------------------------------------
	// initializing local databases
	//---------------------------------------------------------------------------
	d.logger.Info("initializing domain storage", zap.String("id", d.StringID()))

	// general database
	db, err := initLocalDatabase(filepath.Join(d.StoragePath(), fmt.Sprintf("%s.dat", d.StorageID())))
	if err != nil {
		return err
	}

	// password database
	pdb, err := initLocalPasswordDatabase(filepath.Join(d.StoragePath(), "passwords"))
	if err != nil {
		return err
	}

	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	d.logger.Info("initializing domain stores", zap.String("id", d.StringID()))

	us, err := NewDefaultUserStore(db, NewUserStoreCache(1000))
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %s", err)
	}

	gs, err := NewDefaultGroupStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize group store: %s", err)
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
	d.logger.Info("initializing user container", zap.String("did", d.StringID()))
	uc, err := NewUserContainer(us)
	if err != nil {
		return fmt.Errorf("failed to initialize user container: %s", err)
	}

	if err := uc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate user container: %s", err)
	}

	d.logger.Info("initializing group container", zap.String("did", d.StringID()))
	gc, err := NewGroupContainer(gs)
	if err != nil {
		return fmt.Errorf("failed to initialize group container: %s", err)
	}

	if err := gc.Validate(); err != nil {
		return fmt.Errorf("Domain.Init() failed to validate group container: %s", err)
	}

	if err := d.saveConfig(); err != nil {
		return fmt.Errorf("failed to initialize domain configuration: %s", err)
	}

	//---------------------------------------------------------------------------
	// initializing the user password manager
	//---------------------------------------------------------------------------
	pm, err := NewDefaultPasswordManager(ps)
	if err != nil {
		return fmt.Errorf("failed to initialize user password manager: %s", err)
	}

	//---------------------------------------------------------------------------
	// finalizing domain initialization
	//---------------------------------------------------------------------------
	d.Users = uc
	d.Groups = gc
	d.Passwords = pm

	return nil
}

// StringID is a short text info
func (d *Domain) StringID() string {
	return fmt.Sprintf("%s", d.ID)
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
