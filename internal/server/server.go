package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"gitlab.com/agubarev/hometown/auth"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/go-chi/chi"

	"go.etcd.io/bbolt"

	"github.com/spf13/viper"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
}

// StartHometownServer starts the main server
func StartHometownServer() error {
	//---------------------------------------------------------------------------
	// loading and validating configuration
	//---------------------------------------------------------------------------
	log.Println("loading instance configuration")
	storageBackend := viper.GetString("instance.storage.backend")
	dbfile := viper.GetString("instance.storage.dbfile")

	// basic check, only supporting bbolt for now
	if strings.TrimSpace(storageBackend) == "" {
		return fmt.Errorf("backend storage is not specified")
	}

	// making sure database file is set
	if strings.TrimSpace(dbfile) == "" {
		return fmt.Errorf("database file is not specified")
	}

	// loading the database
	log.Printf("loading database [%s]", dbfile)
	db, err := bbolt.Open(dbfile, 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open bbolt database: %s", err)
	}
	defer db.Close()

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

	log.Println("initializing the user manager")
	m := usermanager.New()
	if err != nil {
		return fmt.Errorf("failed to create a new user manager instance: %s", err)
	}

	// initializing the manager; Init() also performs config validation
	err = m.Init(usermanager.NewConfig(usermanager.NewStore(ds, us, gs, aps)))
	if err != nil {
		return fmt.Errorf("failed to initialize the user manager: %s", err)
	}

	//---------------------------------------------------------------------------
	// initializing routes and starting the server
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
