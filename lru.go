package memory

import (
	"container/list"
	"errors"
	"golang.org/x/sync/singleflight"
	"sync"
	"sync/atomic"
	"time"
)

type LruCache interface {
	Cache

	initCache()

	Contains(key string) bool

	// Removes the oldest entry from cache.
	RemoveOldest()

	// Returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (key string, value interface{}, ok bool)

	// Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []string

	Range(f func(key string, value interface{}) error) error
}

type (
	// LruOption defines the method to customize a LruMemory.
	LruOption func(cache *LruMemory)

	// handleFunc
	handleFunc func(key string, value interface{}) error

	LruMemory struct {
		init bool

		name    string
		lock    sync.RWMutex
		atomic  atomic.Value
		barrier singleflight.Group
		size    int
		queue   *list.List
		items   map[interface{}]*list.Element
		stat    *CacheStat

		initLru  initLru
		flushLru flushLru
		onRemove handleFunc
	}

	// entry is used to hold a value in the list
	entry struct {
		key   interface{}
		value interface{}
	}

	initLru struct {
		flag     int32
		onInit   func()
		interval time.Duration
	}

	flushLru struct {
		flag     int32
		onFlush  handleFunc
		interval time.Duration
	}
)

func WithSize(size int) LruOption {
	return func(lru *LruMemory) {
		lru.size = size
	}
}

func WithInit(f func()) LruOption {
	return func(lru *LruMemory) {
		lru.initLru.onInit = f
	}
}

func WithOnRemove(f handleFunc) LruOption {
	return func(lru *LruMemory) {
		lru.onRemove = f
	}
}

func WithFlush(f handleFunc) LruOption {
	return func(lru *LruMemory) {
		lru.flushLru.onFlush = f
	}
}

func WithInitInterval(t time.Duration) LruOption {
	return func(lru *LruMemory) {
		lru.initLru.interval = t
	}
}

func WithFlushInterval(t time.Duration) LruOption {
	return func(lru *LruMemory) {
		lru.flushLru.interval = t
	}
}

func instanceLru(name string, opts ...LruOption) (*LruMemory, error) {
	lru := &LruMemory{
		name:  name,
		size:  10 * 1024,
		queue: list.New(),
		items: make(map[interface{}]*list.Element),
		stat:  &CacheStat{},
	}

	for _, opt := range opts {
		opt(lru)
	}

	if lru.size == 0 {
		return nil, errors.New("must provide a positive size")
	}

	lru.start()
	return lru, nil
}

func (c *LruMemory) start() {
	c.initCache()
	c.flushCache()
}

func (c *LruMemory) Add(key string, value interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	if ent, ok := c.items[key]; ok {
		c.queue.MoveToFront(ent)
		return true
	}

	// Add new item
	ent := &entry{
		key:   key,
		value: value,
	}
	elem := c.queue.PushFront(ent)
	c.items[key] = elem
	if c.queue.Len() > c.size {
		c.RemoveOldest()
	}
	return true
}

func (c *LruMemory) Get(key string) (value interface{}, ok bool) {
	ent, ok := c.doGet(key)
	if ok {
		c.stat.IncrementHit()
	} else {
		c.stat.IncrementMiss()
	}
	return ent.value, ok
}

func (c *LruMemory) doGet(key string) (*entry, bool) {
	c.lock.RLock()
	v, ok := c.items[key]
	c.lock.RUnlock()
	if ok {
		val := v.Value.(*entry)
		c.Add(key, val)
		return val, true
	}
	return nil, false
}

func (c *LruMemory) Take(key string, f func() (interface{}, error)) (value interface{}, err error) {
	if val, ok := c.doGet(key); ok {
		c.stat.IncrementHit()
		return val, nil
	}

	var task bool
	value, err, _ = c.barrier.Do(key, func() (interface{}, error) {
		if val, ok := c.doGet(key); ok {
			c.stat.IncrementHit()
			return val, nil
		}

		var val interface{}
		val, err = f()
		if err != nil {
			return nil, err
		}

		if c.Add(key, val) {
			task = true
		}
		return val, nil
	})

	return value, err
}

func (c *LruMemory) GetOldest() (key string, value interface{}, ok bool) {
	ent := c.queue.Back()
	if ent != nil {
		elems, exist := ent.Value.(*entry)
		if ok {
			return elems.key.(string), elems.value, exist
		}
	}

	return "", nil, false
}

func (c *LruMemory) Len() int {
	return c.queue.Len()
}

func (c *LruMemory) Remove(key string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if elem, ok := c.items[key]; ok {
		c.queue.Remove(elem)
		delete(c.items, elem)

		ent := elem.Value.(*entry)
		if c.onRemove != nil {
			_ = c.onRemove(ent.key.(string), ent.value)
		}
	}
	return true
}

func (c *LruMemory) RemoveOldest() {
	ent := c.queue.Back()
	if ent != nil {
		elems := ent.Value.(*entry)
		c.Remove(elems.key.(string))
	}
}

func (c *LruMemory) Contains(key string) (ok bool) {
	c.lock.RLock()
	defer c.lock.RLock()
	_, ok = c.items[key]
	return
}

func (c *LruMemory) Stat() *CacheStat {
	return c.stat
}

func (c *LruMemory) Keys() []string {
	keys := make([]string, 0, len(c.items))
	_ = c.Range(func(key string, value interface{}) error {
		keys = append(keys, key)
		return nil
	})
	return keys
}

func (c *LruMemory) Range(f func(key string, value interface{}) error) error {
	for ent := c.queue.Back(); ent != nil; ent = ent.Prev() {
		val := ent.Value.(*entry)
		err := f(val.key.(string), val.value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *LruMemory) initCache() {
	if c.initLru.interval > 0 && c.initLru.onInit != nil {
		if !c.init { // 防止多次初始化
			c.initLru.onInit()
			c.init = true
		}

		if !atomic.CompareAndSwapInt32(&c.initLru.flag, 0, 1) { //防止同时运行多个
			return
		}

		go func() {
			t := time.NewTicker(c.initLru.interval)
			for {
				select {
				case <-t.C:
					c.initLru.onInit()
					atomic.StoreInt32(&c.initLru.flag, 0)
				default:
				}
			}
		}()
	}
}

func (c *LruMemory) flushCache() {
	if c.flushLru.interval > 0 && c.flushLru.onFlush != nil {
		if !atomic.CompareAndSwapInt32(&c.flushLru.flag, 0, 1) { //防止同时运行多个
			return
		}

		go func() {
			t := time.NewTicker(c.flushLru.interval)
			for {
				select {
				case <-t.C:
					_ = c.Range(c.flushLru.onFlush)
					atomic.StoreInt32(&c.flushLru.flag, 0)
				default:
				}
			}
		}()
	}
}

func (s *CacheStat) IncrementHit() {
	atomic.AddUint64(&s.hit, 1)
}

func (s *CacheStat) IncrementMiss() {
	atomic.AddUint64(&s.miss, 1)
}
