package auth

import (
	"context"

	"github.com/allegro/bigcache"
	"github.com/pkg/errors"
)

type Cache interface {
	Put(ctx context.Context, key string, entry []byte) (err error)
	Get(ctx context.Context, key string) (entry []byte, err error)
	Delete(ctx context.Context, key string) (err error)
}

type defaultCache struct {
	backend *bigcache.BigCache
}

func NewDefaultCache(config bigcache.Config) (Cache, error) {
	backend, err := bigcache.NewBigCache(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize default cache")
	}

	cache := &defaultCache{
		backend: backend,
	}

	return cache, nil
}

func (d *defaultCache) Put(ctx context.Context, key string, entry []byte) (err error) {
	return errors.Wrapf(d.backend.Set(key, entry), "failed to cache entry %s -> [%v]", key, entry)
}

func (d *defaultCache) Get(ctx context.Context, key string) (entry []byte, err error) {
	panic("implement me")
}

func (d *defaultCache) Delete(ctx context.Context, key string) (err error) {
	panic("implement me")
}
