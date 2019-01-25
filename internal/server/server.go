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

	"github.com/spf13/viper"
)

type contextKey string

func (k contextKey) String() string {
	return string(k)
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
