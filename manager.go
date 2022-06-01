package memory

import (
	"context"
	"errors"
	"memory/cache"
	"sync"
	"time"
)

type (
	Manager struct {
		ctx       context.Context
		ctxCancel context.CancelFunc
		M         sync.Map
	}

	Entry struct {
		ctx     context.Context
		name    string
		memory  cache.Cache
		preload bool
		channel chan func(cache cache.Cache)
		fresh   *Handle
		flush   *Handle
	}

	EntryOption func(ent *Entry)

	Handle struct {
		flag     bool
		handle   func(cache cache.Cache)
		interval time.Duration
	}
)

func WithFresh(f func(cache cache.Cache), duration time.Duration, preload bool) EntryOption {
	return func(ent *Entry) {
		ent.fresh = &Handle{
			flag:     false,
			handle:   f,
			interval: duration,
		}
		ent.preload = preload
	}
}

func WithFlush(f func(cache cache.Cache), duration time.Duration) EntryOption {
	return func(ent *Entry) {
		ent.flush = &Handle{
			flag:     false,
			handle:   f,
			interval: duration,
		}
	}
}

func New() *Manager {
	c, cancelFunc := context.WithCancel(context.Background())
	return &Manager{
		ctx:       c,
		ctxCancel: cancelFunc,
		M:         sync.Map{},
	}
}

func (m *Manager) Add(name string, cache cache.Cache, opts ...EntryOption) error {
	if _, ok := m.M.Load(name); ok {
		return errors.New("cache entry is already exist")
	}

	ent := &Entry{
		ctx:     m.ctx,
		name:    name,
		memory:  cache,
		preload: false,
	}

	for _, opt := range opts {
		opt(ent)
	}

	go ent.run()
	m.M.Store(name, ent)
	return nil
}

func (e *Entry) run() {
	if e.fresh != nil {
		if e.preload {
			e.fresh.handle(e.memory)
		}
		if e.fresh.interval > 0 {
			ticker := time.NewTicker(e.fresh.interval)
		FRESH:
			for {
				select {
				case <-ticker.C:
					e.channel <- e.fresh.handle
				case <-e.ctx.Done():
					break FRESH
				default:
				}
			}
		}
	}

	if e.flush != nil && e.flush.interval > 0 {
		ticker := time.NewTicker(e.flush.interval)
	FLUSH:
		for {
			select {
			case <-ticker.C:
				e.channel <- e.flush.handle
			case <-e.ctx.Done():
				break FLUSH
			default:
			}
		}
	}

	go e.task()
}

func (e *Entry) task() {
TASK:
	for {
		select {
		case f := <-e.channel:
			go f(e.memory)
		case <-e.ctx.Done():
			break TASK
		default:
		}
	}
}

func (m *Manager[K, V]) Close() {
	m.ctxCancel()

	wait := sync.WaitGroup{}
	m.M.Range(func(key, value any) bool {
		wait.Add(1)
		go func(ent *Entry) {
			close(ent.channel)
			if ent.flush != nil {
				ent.flush.handle(ent.memory)
			}
			defer wait.Done()
		}(value.(*Entry))
		return true
	})
	wait.Wait()
}
