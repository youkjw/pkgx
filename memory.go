package memory

import (
	"errors"
	"sync"
)

type (
	CacheStat struct {
		miss, hit uint64
	}

	Cache interface {
		Add(key string, value interface{}) bool
		Get(key string) (interface{}, bool)
		Take(key string, f func() (interface{}, error)) (interface{}, error)
		Remove(key string) bool
		Len() int
		Stat() *CacheStat
		flushCache()
	}
)

var (
	cacheMap = sync.Map{}
)

func Lru(name string, opts ...LruOption) (*LruMemory, error) {
	if c, ok := cacheMap.Load(name); ok {
		if _, ok = c.(*LruMemory); ok {
			return c.(*LruMemory), nil
		}
		return nil, errors.New("memory already exists but not lru")
	}
	cache, err := instanceLru(name, opts...)
	cacheMap.Store(name, cache)
	return cache, err
}

func Close() {
	wait := sync.WaitGroup{}
	cacheMap.Range(func(key, value interface{}) bool {
		wait.Add(1)
		go func(c Cache) {
			defer wait.Done()
			c.flushCache()
		}(value.(Cache))
		return true
	})
	wait.Wait()
}
