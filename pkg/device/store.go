package device

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type Store interface {
	UpsertDevice(ctx context.Context, d Device) (Device, error)
	CreateRelation(ctx context.Context, rel Relation) error
	HasRelation(ctx context.Context, rel Relation) bool
	FetchDeviceByID(ctx context.Context, deviceID uuid.UUID) (d Device, err error)
	FetchAllDevices(ctx context.Context) (ds []Device, err error)
	FetchAllRelations(ctx context.Context) ([]Relation, error)
	DeleteDeviceByID(ctx context.Context, deviceID uuid.UUID) error
	DeleteRelation(ctx context.Context, rel Relation) error
}

type memoryStore struct {
	devices   map[uuid.UUID]Device
	relations []Relation
	sync.RWMutex
}

func NewMemoryStore() Store {
	return &memoryStore{
		devices:   make(map[uuid.UUID]Device),
		relations: make([]Relation, 0),
	}
}

func (s *memoryStore) UpsertDevice(ctx context.Context, d Device) (Device, error) {
	s.Lock()
	s.devices[d.ID] = d
	s.Unlock()

	return d, nil
}

func (s *memoryStore) CreateRelation(ctx context.Context, rel Relation) error {
	s.RLock()
	for _, r := range s.relations {
		if r == rel {
			s.RUnlock()
			return ErrRelationAlreadyExists
		}
	}
	s.RUnlock()

	s.Lock()
	s.relations = append(s.relations, rel)
	s.Unlock()

	return nil
}

func (s *memoryStore) FetchDeviceByID(ctx context.Context, deviceID uuid.UUID) (d Device, err error) {
	s.RLock()
	d, ok := s.devices[deviceID]
	s.RUnlock()

	if ok {
		return d, nil
	}

	return d, ErrDeviceNotFound
}

func (s *memoryStore) HasRelation(ctx context.Context, rel Relation) bool {
	s.RLock()
	for _, r := range s.relations {
		if r == rel {
			return true
		}
	}
	s.RUnlock()

	return false
}

func (s *memoryStore) FetchAllDevices(ctx context.Context) (ds []Device, err error) {
	ds = make([]Device, 0, len(s.devices))

	s.RLock()
	for id := range s.devices {
		ds = append(ds, s.devices[id])
	}
	s.RUnlock()

	return ds, nil
}

func (s *memoryStore) FetchAllRelations(ctx context.Context) (rels []Relation, err error) {
	rels = make([]Relation, 0, len(s.relations))

	s.RLock()
	for i := range s.relations {
		rels = append(rels, s.relations[i])
	}
	s.RUnlock()

	return rels, nil
}

func (s *memoryStore) DeleteDeviceByID(ctx context.Context, deviceID uuid.UUID) error {
	s.Lock()
	delete(s.devices, deviceID)
	s.Unlock()

	return nil
}

func (s *memoryStore) DeleteRelation(ctx context.Context, rel Relation) error {
	s.Lock()
	for i, r := range s.relations {
		if r == rel {
			s.relations = append(s.relations[:i], s.relations[i+1:]...)
			return nil
		}
	}
	s.Unlock()

	return nil
}
