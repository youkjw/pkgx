package memory

import (
	"container/list"
	"context"
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

	flushCache()
}

type (
	// LruOption defines the method to customize a lruMemory.
	LruOption func(cache *lruMemory)

	// handleFunc
	handleFunc func(key string, value interface{}) error
	// lruFunc
	lruFunc func(memory *lruMemory) error

	lruMemory struct {
		inited bool

		ctx       context.Context
		ctxCancel context.CancelFunc

		name    string
		size    int
		lock    sync.RWMutex
		atomic  atomic.Value
		barrier singleflight.Group
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
		onInit   lruFunc
		interval time.Duration
	}

	flushLru struct {
		flag     int32
		onFlush  handleFunc
		interval time.Duration
	}
)

func WithSize(size int) LruOption {
	return func(lru *lruMemory) {
		lru.size = size
	}
}

func WithInit(f func(*lruMemory) error) LruOption {
	return func(lru *lruMemory) {
		lru.initLru.onInit = f
	}
}

func WithOnRemove(f handleFunc) LruOption {
	return func(lru *lruMemory) {
		lru.onRemove = f
	}
}

func WithFlush(f handleFunc) LruOption {
	return func(lru *lruMemory) {
		lru.flushLru.onFlush = f
	}
}

func WithInitInterval(t time.Duration) LruOption {
	return func(lru *lruMemory) {
		lru.initLru.interval = t
	}
}

func WithFlushInterval(t time.Duration) LruOption {
	return func(lru *lruMemory) {
		lru.flushLru.interval = t
	}
}

func instanceLru(name string, opts ...LruOption) (*lruMemory, error) {
	lru := &lruMemory{
		name:  name,
		size:  0,
		queue: list.New(),
		items: make(map[interface{}]*list.Element),
		stat:  &CacheStat{},
	}

	for _, opt := range opts {
		opt(lru)
	}

	lru.ctx, lru.ctxCancel = context.WithCancel(context.Background())
	lru.start()
	return lru, nil
}

func (c *lruMemory) start() {
	c.initCache()
	c.flushCache()
}

func (c *lruMemory) Add(key string, value interface{}) bool {
	c.lock.Lock()
	if ent, ok := c.items[key]; ok {
		ent.Value.(*entry).value = value
		c.queue.MoveToFront(ent)
		c.lock.Unlock()
		return true
	}

	// Add new item
	ent := &entry{
		key:   key,
		value: value,
	}
	elem := c.queue.PushFront(ent)
	c.items[key] = elem
	c.lock.Unlock()
	if c.size > 0 && c.queue.Len() > c.size {
		c.RemoveOldest()
	}
	return true
}

func (c *lruMemory) Get(key string) (value interface{}, ok bool) {
	ent, ok := c.doGet(key)
	if ok {
		c.stat.IncrementHit()
	} else {
		c.stat.IncrementMiss()
		return nil, false
	}
	return ent.value, ok
}

func (c *lruMemory) doGet(key string) (*entry, bool) {
	c.lock.RLock()
	v, ok := c.items[key]
	c.lock.RUnlock()
	if ok {
		val := v.Value.(*entry)
		c.Add(key, val.value)
		return val, true
	}
	return nil, false
}

func (c *lruMemory) Take(key string, f func() (interface{}, error)) (value interface{}, err error) {
	if val, ok := c.doGet(key); ok {
		c.stat.IncrementHit()
		return val.value, nil
	}

	var task bool
	value, err, _ = c.barrier.Do(key, func() (interface{}, error) {
		if val, ok := c.doGet(key); ok {
			return val.value, nil
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

	if task {
		c.stat.IncrementMiss()
		return value, err
	}

	c.stat.IncrementHit()
	return value, err
}

func (c *lruMemory) GetOldest() (key string, value interface{}, ok bool) {
	ent := c.queue.Back()
	if ent != nil {
		elems, exist := ent.Value.(*entry)
		if ok {
			return elems.key.(string), elems.value, exist
		}
	}

	return "", nil, false
}

func (c *lruMemory) Len() int {
	return c.queue.Len()
}

func (c *lruMemory) Remove(key string) bool {
	c.lock.Lock()
	if elem, ok := c.items[key]; ok {
		c.queue.Remove(elem)
		delete(c.items, key)

		ent := elem.Value.(*entry)
		c.lock.Unlock()
		if c.onRemove != nil {
			_ = c.onRemove(ent.key.(string), ent.value)
		}
		return true
	}
	c.lock.Unlock()
	return true
}

func (c *lruMemory) RemoveOldest() {
	ent := c.queue.Back()
	if ent != nil {
		elems := ent.Value.(*entry)
		c.Remove(elems.key.(string))
	}
}

func (c *lruMemory) Contains(key string) (ok bool) {
	c.lock.RLock()
	defer c.lock.RLock()
	_, ok = c.items[key]
	return
}

func (c *lruMemory) Stat() *CacheStat {
	return c.stat
}

func (c *lruMemory) Keys() []string {
	keys := make([]string, 0, len(c.items))
	_ = c.Range(func(key string, value interface{}) error {
		keys = append(keys, key)
		return nil
	})
	return keys
}

func (c *lruMemory) Range(f func(key string, value interface{}) error) error {
	for ent := c.queue.Back(); ent != nil; ent = ent.Prev() {
		val := ent.Value.(*entry)
		err := f(val.key.(string), val.value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *lruMemory) initCache() {
	if c.initLru.interval > 0 && c.initLru.onInit != nil {
		if !c.inited { // 防止多次初始化
			c.inited = true
			_ = c.initLru.onInit(c)
		}

		if !atomic.CompareAndSwapInt32(&c.initLru.flag, 0, 1) { //防止同时运行多个
			return
		}

		go func() {
			t := time.NewTicker(c.initLru.interval)
		INIT:
			for {
				select {
				case <-t.C:
					_ = c.initLru.onInit(c)
					atomic.StoreInt32(&c.initLru.flag, 0)
				case <-c.ctx.Done():
					break INIT
				}
			}
		}()
	}
}

func (c *lruMemory) flushCache() {
	if c.flushLru.interval > 0 && c.flushLru.onFlush != nil {
		if !atomic.CompareAndSwapInt32(&c.flushLru.flag, 0, 1) { //防止同时运行多个
			return
		}

		go func() {
			t := time.NewTicker(c.flushLru.interval)
		FLUSH:
			for {
				select {
				case <-t.C:
					_ = c.Range(c.flushLru.onFlush)
					atomic.StoreInt32(&c.flushLru.flag, 0)
				case <-c.ctx.Done():
					break FLUSH
				}
			}
		}()
	}
}

func (c *lruMemory) Close() {
	c.ctxCancel()
	if c.flushLru.onFlush != nil {
		_ = c.Range(c.flushLru.onFlush)
	}
}

func (s *CacheStat) IncrementHit() {
	atomic.AddUint64(&s.hit, 1)
}

func (s *CacheStat) IncrementMiss() {
	atomic.AddUint64(&s.miss, 1)
}
