package client

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Store interface {
	UpsertClient(ctx context.Context, c Client) (Client, error)
	FetchClientByID(ctx context.Context, clientID uuid.UUID) (c Client, err error)
	FetchAllClients(ctx context.Context) (cs []Client, err error)
	DeleteClientByID(ctx context.Context, clientID uuid.UUID) error
}

type memoryStore struct {
	clients map[uuid.UUID]Client
	sync.RWMutex
}

func NewMemoryStore() Store {
	return &memoryStore{
		clients: make(map[uuid.UUID]Client),
	}
}

func (s *memoryStore) UpsertClient(ctx context.Context, c Client) (Client, error) {
	s.Lock()
	s.clients[c.ID] = c
	s.Unlock()

	return c, nil
}

func (s *memoryStore) FetchClientByID(ctx context.Context, clientID uuid.UUID) (c Client, err error) {
	s.RLock()
	c, ok := s.clients[clientID]
	s.RUnlock()

	if ok {
		return c, nil
	}

	return c, ErrClientNotFound
}
func (s *memoryStore) FetchAllClients(ctx context.Context) (cs []Client, err error) {
	cs = make([]Client, 0, len(s.clients))

	s.RLock()
	for id := range s.clients {
		cs = append(cs, s.clients[id])
	}
	s.RUnlock()

	return cs, nil
}
func (s *memoryStore) DeleteClientByID(ctx context.Context, clientID uuid.UUID) error {
	s.Lock()
	delete(s.clients, clientID)
	s.Unlock()

	return nil
}
