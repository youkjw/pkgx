package lru

import (
	"container/list"
	"context"
	"golang.org/x/sync/singleflight"
	"pkgx/cache"
	"sync"
	"sync/atomic"
)

type (
	// Option defines the method to customize a Lru.
	Option func(lru *lru)

	// handleFunc
	handleFunc func(key string, value interface{}) error

	lru struct {
		ctx       context.Context
		ctxCancel context.CancelFunc

		name    string
		size    int
		lock    sync.RWMutex
		atomic  atomic.Value
		barrier singleflight.Group
		queue   *list.List
		items   map[interface{}]*list.Element
		stat    *cache.CacheStat

		onRemove handleFunc
	}

	// entry is used to hold a value in the list
	entry struct {
		key   interface{}
		value interface{}
	}
)

func WithSize(size int) Option {
	return func(lru *lru) {
		lru.size = size
	}
}

func WithOnRemove(f handleFunc) Option {
	return func(lru *lru) {
		lru.onRemove = f
	}
}

func NewLru(name string, opts ...Option) (*lru, error) {
	Lru := &lru{
		name:  name,
		size:  0,
		queue: list.New(),
		items: make(map[interface{}]*list.Element),
		stat:  &cache.CacheStat{},
	}

	for _, opt := range opts {
		opt(Lru)
	}

	Lru.ctx, Lru.ctxCancel = context.WithCancel(context.Background())
	return Lru, nil
}

func (c *lru) Add(key string, value interface{}) bool {
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

func (c *lru) Get(key string) (value interface{}, ok bool) {
	ent, ok := c.doGet(key)
	if ok {
		c.stat.IncrementHit()
	} else {
		c.stat.IncrementMiss()
		return nil, false
	}
	return ent.value, ok
}

func (c *lru) doGet(key string) (*entry, bool) {
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

func (c *lru) Take(key string, f func() (interface{}, error)) (value interface{}, err error) {
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

func (c *lru) GetOldest() (key string, value interface{}, ok bool) {
	ent := c.queue.Back()
	if ent != nil {
		elems, exist := ent.Value.(*entry)
		if ok {
			return elems.key.(string), elems.value, exist
		}
	}

	return "", nil, false
}

func (c *lru) Len() int {
	return c.queue.Len()
}

func (c *lru) Remove(key string) bool {
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

func (c *lru) RemoveOldest() {
	ent := c.queue.Back()
	if ent != nil {
		elems := ent.Value.(*entry)
		c.Remove(elems.key.(string))
	}
}

func (c *lru) Contains(key string) (ok bool) {
	c.lock.RLock()
	defer c.lock.RLock()
	_, ok = c.items[key]
	return
}

func (c *lru) Stat() *cache.CacheStat {
	return c.stat
}

func (c *lru) Keys() []string {
	keys := make([]string, 0, len(c.items))
	_ = c.Range(func(key string, value interface{}) error {
		keys = append(keys, key)
		return nil
	})
	return keys
}

func (c *lru) Range(f func(key string, value interface{}) error) error {
	for ent := c.queue.Back(); ent != nil; ent = ent.Prev() {
		val := ent.Value.(*entry)
		err := f(val.key.(string), val.value)
		if err != nil {
			return err
		}
	}
	return nil
}
