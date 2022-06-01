package cache

import (
	"container/list"
	"context"
	"golang.org/x/sync/singleflight"
	"sync"
	"sync/atomic"
)

type (
	// LruOption defines the method to customize a lruMemory.
	LruOption func(lru *lruMemory)

	// handleFunc
	handleFunc func(key string, value interface{}) error

	lruMemory struct {
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

		onRemove handleFunc
	}

	// entry is used to hold a value in the list
	entry struct {
		key   interface{}
		value interface{}
	}
)

func WithSize(size int) LruOption {
	return func(lru *lruMemory) {
		lru.size = size
	}
}

func WithOnRemove(f handleFunc) LruOption {
	return func(lru *lruMemory) {
		lru.onRemove = f
	}
}

func NewLru(name string, opts ...LruOption) (*lruMemory, error) {
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
	return lru, nil
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

func (s *CacheStat) IncrementHit() {
	atomic.AddUint64(&s.hit, 1)
}

func (s *CacheStat) IncrementMiss() {
	atomic.AddUint64(&s.miss, 1)
}
