package usermanager

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blevesearch/bleve"

	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
	"github.com/spf13/viper"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

// DomainConfig is a configuration object for a domain
type DomainConfig struct {
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

	CreatedAt   time.Time `json:"t_cr"`
	UpdatedAt   time.Time `json:"t_up,omitempty"`
	ConfirmedAt time.Time `json:"t_co,omitempty"`

	store DomainStore
	index bleve.Index
	sync.RWMutex
}

func localStorageDirectory() string {
	return viper.GetString("instance.domains.directory")
}

func initLocalStorageDirectory() error {
	// domain storage directory must be set
	dsdir := localStorageDirectory()
	if dsdir == "" {
		return fmt.Errorf("local domain storage directory is not defined")
	}

	// domain storage directory must exist during validation
	fstat, err := os.Stat(dsdir)
	if err != nil {
		if os.IsNotExist(err) {
			// attempting to create storage directory
			if err := os.MkdirAll(dsdir, 0600); err != nil {
				return fmt.Errorf("failed to create local domain storage directory: %s", err)
			}
		}

		return fmt.Errorf("failed to initialize local storage directory: %s", err)
	}

	// domain storage must be readable by the instance
	if _, err := ioutil.ReadDir(dsdir); err != nil {
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

	db, err := bbolt.Open(dbfile, 0600, nil)
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
	fstat, err := os.Stat(dbDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0600); err != nil {
			return nil, fmt.Errorf("failed to create local password database directory: %s", err)
		}
	}

	// path must be a directory
	if !fstat.Mode().IsDir() {
		return nil, fmt.Errorf("given password database directory path [%s] is not a directory", dbDir)
	}

	// attempting to open password database
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open local passwords database: %s", err)
	}

	return db, nil
}

// NewDomain initializing new domain
func NewDomain(owner *User) (*Domain, error) {
	// each new domain must have an owner user assigned
	if owner == nil {
		return nil, fmt.Errorf("new domain must have an owner: %s", ErrNilUser)
	}

	// new domain
	domain := &Domain{
		ID: util.NewULID(),

		// initial domain owner, full access to this domain
		Owner: owner,

		// at this moment each policy is independent by default
		// it doesn't have a parent by default, doesn't inherit nor extends anything
		// TODO think through the default initial state of an AccessPolicy
		// TODO: add domain ID as policy ID
		AccessPolicy: NewAccessPolicy(owner, nil, false, false),
	}

	//---------------------------------------------------------------------------
	// initializing local storage
	//---------------------------------------------------------------------------
	// precautious check
	if err := initLocalStorageDirectory(); err != nil {
		return nil, err
	}

	// creating storage if it hasn't been created yet
	sdir := domain.StorageDir()
	if _, err := os.Stat(sdir); err != nil {
		if os.IsNotExist(err) {
			// not found, attempting to create
			if err := os.MkdirAll(sdir, 0600); err != nil {
				return nil, fmt.Errorf("failed to create domain storage: %s")
			}
		} else {
			return nil, fmt.Errorf("failed to stat domain storage: %s", err)
		}
	}

	//---------------------------------------------------------------------------
	// initializing local databases
	//---------------------------------------------------------------------------
	// common database
	db, err := initLocalDatabase(filepath.Join(domain.StorageDir(), domain.StorageID()))
	if err != nil {
		return nil, err
	}

	// password database
	pdb, err := initLocalPasswordDatabase(filepath.Join(domain.StorageDir(), "passwords", domain.StorageID()))
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := NewDefaultUserStore(db, NewUserStoreCache(1000))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user store: %s", err)
	}

	gs, err := NewDefaultGroupStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize group store: %s", err)
	}

	aps, err := NewDefaultAccessPolicyStore(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize access policy store: %s", err)
	}

	ps, err := NewDefaultPasswordStore(pdb)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize password store: %s", err)
	}

	//---------------------------------------------------------------------------
	// initializing and validating containers
	//---------------------------------------------------------------------------
	uc, err := NewUserContainer(us)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user container: %s", err)
	}

	if err := uc.Validate(); err != nil {
		return nil, fmt.Errorf("Domain.Init() failed to validate user container: %s", err)
	}

	gc, err := NewGroupContainer(gs)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize group container: %s", err)
	}

	if err := gc.Validate(); err != nil {
		return nil, fmt.Errorf("Domain.Init() failed to validate group container: %s", err)
	}

	//---------------------------------------------------------------------------
	// finalizing domain initialization
	//---------------------------------------------------------------------------
	domain.Users = uc
	domain.Groups = gc

	return domain, nil
}

// IDString is a short text info
func (d *Domain) IDString() string {
	return fmt.Sprintf("domain[%s]", d.ID)
}

// StorageID returns a string which should be used to identify
// this domain inside a respective storage, i.e. directory name
// inside a local domain storage directory
func (d *Domain) StorageID() string {
	return fmt.Sprintf("%s", d.ID)
}

// StorageDir returns domain's local storage directory path
func (d *Domain) StorageDir() string {
	return filepath.Join(localStorageDirectory(), d.StorageID())
}
