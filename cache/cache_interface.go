package cache

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
		Contains(key string) bool
		// Removes the oldest entry from cache.
		RemoveOldest()
		// Returns the oldest entry from the cache. #key, value, isFound
		GetOldest() (key string, value interface{}, ok bool)
		// Returns a slice of the keys in the cache, from oldest to newest.
		Keys() []string
		Range(f func(key string, value interface{}) error) error
	}
)
