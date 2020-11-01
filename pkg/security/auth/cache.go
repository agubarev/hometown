package auth

import (
	"context"

	"github.com/allegro/bigcache"
	"github.com/pkg/errors"
)

type Cache interface {
	Put(ctx context.Context, key string, entry []byte) (err error)
	Get(ctx context.Context, key string) (entry []byte, err error)
	Take(ctx context.Context, key string) (entry []byte, err error)
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
	return errors.Wrapf(d.backend.Set(key, entry), "failed to cache entry: %s => [%v", key, entry)
}

// Get returns a copy of the stored value
func (d *defaultCache) Get(ctx context.Context, key string) (entry []byte, err error) {
	entry, err = d.backend.Get(key)
	if err != nil {
		if err == bigcache.ErrEntryNotFound {
			return nil, ErrEntryNotFound
		}

		return nil, errors.Wrapf(err, "failed to obtain cached entry: %s", key)
	}

	return entry, nil
}

// Take works like Get except that it deletes the key after reading
func (d *defaultCache) Take(ctx context.Context, key string) (entry []byte, err error) {
	entry, err = d.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if err = d.Delete(ctx, key); err != nil {
		return nil, err
	}

	return entry, nil
}

func (d *defaultCache) Delete(ctx context.Context, key string) (err error) {
	if err = d.backend.Delete(key); err != nil {
		return errors.Wrapf(err, "failed to delete cached entry: %s", key)
	}

	return nil
}
