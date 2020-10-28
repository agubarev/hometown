package password

import (
	"context"
	"sync"
)

type Store interface {
	Upsert(ctx context.Context, p Password) error
	Get(ctx context.Context, o Owner) (Password, error)
	Delete(ctx context.Context, o Owner) error
}

func NewMemoryStore() Store {
	return &memoryStore{
		passwords: make(map[Owner]Password),
	}
}

type memoryStore struct {
	passwords map[Owner]Password
	sync.RWMutex
}

func (m *memoryStore) Upsert(ctx context.Context, p Password) error {
	m.Lock()
	m.passwords[p.Owner] = p
	m.Unlock()
	return nil
}

func (m *memoryStore) Get(ctx context.Context, o Owner) (Password, error) {
	m.RLock()
	defer m.RUnlock()
	return m.passwords[o], nil
}

func (m *memoryStore) Delete(ctx context.Context, o Owner) error {
	m.Lock()
	delete(m.passwords, o)
	m.Unlock()
	return nil
}
