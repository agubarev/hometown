package user

import (
	"fmt"
	"sync"

	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/blevesearch/bleve"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	TEmailAddr   = [256]byte
	TUsername    = [32]byte
	TDisplayName = [200]byte
	TIPAddr      = [15]byte
	TName        = [120]byte
	TLanguage    = [2]byte
)

// Manager handles business logic of its underlying objects
// TODO: consider naming first release `Lidia`
type Manager struct {
	passwords password.Manager
	store     Store
	index     bleve.Index
	logger    *zap.Logger
	sync.RWMutex
}

// NewManager returns a new user manager instance
// also initializing by loading necessary data from a given store
func NewManager(s Store, idx bleve.Index) (*Manager, error) {
	if s == nil {
		return nil, errors.Wrap(ErrNilStore, "failed to initialize user manager")
	}

	// initializing the user manager
	m := &Manager{
		store: s,
		index: idx,
	}

	//---------------------------------------------------------------------------
	// setting default dependencies
	// NOTE: these dependencies can be overriden later, just using these as a default
	//---------------------------------------------------------------------------
	// initializing and setting a new container
	userContainer, err := NewContainer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize new container: %s", err)
	}

	err = m.SetUserContainer(userContainer)
	if err != nil {
		return nil, fmt.Errorf("failed to set a default user container: %s", err)
	}

	// using default logger
	logger, err := util.DefaultLogger(true, "")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize default logger: %s", err)
	}

	err = m.SetLogger(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to set default logger: %s", err)
	}

	return m, nil
}

// Store returns store if set
func (m *Manager) Store() (Store, error) {
	if m.store == nil {
		return nil, ErrNilStore
	}

	return m.store, nil
}

// SetLogger assigns a primary logger for the manager
func (m *Manager) SetLogger(logger *zap.Logger) error {
	// if logger is set, then giving it a name
	// to know the log context
	if logger != nil {
		logger = logger.Named("[user]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise
// initializing and returning a new default emergency logger
// NOTE: will panic if it finally fails to obtain a logger
func (m *Manager) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			// having a working logger is crucial, thus must panic() if initialization fails
			panic(errors.Wrap(err, "failed to initialize fallback logger"))
		}

		m.logger = l
	}

	return m.logger
}
