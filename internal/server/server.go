package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"go.etcd.io/bbolt"

	"github.com/spf13/viper"
)

type contextKey string

func (ck contextKey) String() string {
	return string(ck)
}

// StartHometownServer starts the main server
func StartHometownServer() error {
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

	// loading database file
	log.Printf("loading database [%s]", dbfile)
	db, err := bbolt.Open(dbfile, 0600, nil)
	if err != nil {
		return fmt.Errorf("failed to open bbolt database: %s", err)
	}
	defer db.Close()

	// configuring http server
	srv := http.Server{
		Addr: viper.GetString("instance.addr"),
	}

	return nil
}
