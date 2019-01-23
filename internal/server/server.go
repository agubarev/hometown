package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"gitlab.com/agubarev/hometown/auth"
	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/dgraph-io/badger"
	"github.com/go-chi/chi"

	"go.etcd.io/bbolt"

	"github.com/spf13/viper"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
}

func openDefaultDatabase(dbfile string) (*bbolt.DB, error) {
	if strings.TrimSpace(dbfile) == "" {
		return nil, fmt.Errorf("database file is not specified")
	}

	db, err := bbolt.Open(dbfile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %s", err)
	}

	return db, nil
}

func openDefaultPasswordDatabase(dbDir string) (*badger.DB, error) {
	dopts := badger.DefaultOptions
	dopts.Dir = dbDir
	dopts.ValueDir = dbDir

	// password storage directory must exist and be writable
	fstat, err := os.Stat(dbDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0600); err != nil {
			return nil, fmt.Errorf("failed to create password database directory: %s", err)
		}
	}

	// path must be a directory
	if !fstat.Mode().IsDir() {
		return nil, fmt.Errorf("given password database directory path [%s] is not a directory", dbDir)
	}

	// attempting to open password database
	db, err := badger.Open(dopts)
	if err != nil {
		return nil, fmt.Errorf("failed to open passwords database: %s", err)
	}

	return db, nil
}

// StartHometown starts the main server
func StartHometown() error {
	//---------------------------------------------------------------------------
	// loading and validating configuration
	//---------------------------------------------------------------------------
	log.Println("loading instance configuration")
	dbfile := viper.GetString("instance.storage.default")
	passDir := viper.GetString("instance.storage.passwords")

	if strings.TrimSpace(passDir) == "" {
		return fmt.Errorf("password database directory is not specified")
	}

	//---------------------------------------------------------------------------
	// loading databases
	//---------------------------------------------------------------------------
	// loading default database
	log.Printf("loading general database [%s]", dbfile)
	db, err := openDefaultDatabase(dbfile)
	if err != nil {
		return err
	}
	defer db.Close()

	// loading default password database
	log.Printf("loading password database [%s]", dbfile)
	pdb, err := openDefaultPasswordDatabase(passDir)
	if err != nil {
		return err
	}

	//---------------------------------------------------------------------------
	// bootstrapping the user manager
	//---------------------------------------------------------------------------
	log.Println("initializing data stores")
	ds, err := usermanager.NewDefaultDomainStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize domain store: %s", err)
	}

	us, err := usermanager.NewDefaultUserStore(db, usermanager.NewUserStoreCache(1000))
	if err != nil {
		return fmt.Errorf("failed to initialize user store: %s", err)
	}

	gs, err := usermanager.NewDefaultGroupStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize group store: %s", err)
	}

	aps, err := usermanager.NewDefaultAccessPolicyStore(db)
	if err != nil {
		return fmt.Errorf("failed to initialize access policy store: %s", err)
	}

	ps, err := usermanager.NewDefaultPasswordStore(pdb)
	if err != nil {
		return fmt.Errorf("failed to initialize access policy store: %s", err)
	}

	log.Println("initializing the user manager")
	m, err := usermanager.New(usermanager.NewStore(ds, us, gs, aps, ps))
	if err != nil {
		return fmt.Errorf("failed to create a new user manager instance: %s", err)
	}

	// initializing the manager; Init() also performs config validation
	err = m.Init()
	if err != nil {
		if err == usermanager.ErrSuperDomainNotFound {
			// repeat until super domain is created successfully
			for {
				err := interactiveCreateSuperDomain(m)
				if err == nil {
					// super domain should be created at this point
					break
				}
			}
		}

		return fmt.Errorf("failed to initialize the user manager: %s", err)
	}

	//---------------------------------------------------------------------------
	// routing and starting the server
	//---------------------------------------------------------------------------
	r := chi.NewRouter()

	// initializing base context
	ctx := context.Background()

	// default middleware stack
	r.Use(auth.MiddlewareAuth)

	r.Route("/user", func(r chi.Router) {
		r.Get("/{id}", usermanager.HandleGetUser)
		r.Post("/", usermanager.HandlePostUser)
		r.Patch("/", usermanager.HandlePatchUser)
		r.Delete("/{id}", usermanager.HandleDeleteUser)
	})

	// starting the listener
	log.Printf("the server is now listening on %s", viper.GetString("instance.addr"))
	return http.ListenAndServe(viper.GetString("instance.addr"), chi.ServerBaseContext(ctx, r))
}
