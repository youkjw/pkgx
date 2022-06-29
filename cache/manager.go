package cache

import (
	"context"
	"errors"
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
		memory  Cache
		preload bool
		channel chan func(cache Cache)
		fresher *Handle
		flusher *Handle
	}

	EntryOption func(ent *Entry)

	Handle struct {
		flag     bool
		handle   func(cache Cache)
		interval time.Duration
	}
)

func WithFresher(f func(cache Cache), duration time.Duration, preload bool) EntryOption {
	return func(entry *Entry) {
		entry.fresher = &Handle{
			flag:     false,
			handle:   f,
			interval: duration,
		}
		entry.preload = preload
	}
}

func WithFlusher(f func(cache Cache), duration time.Duration) EntryOption {
	return func(ent *Entry) {
		ent.flusher = &Handle{
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

func (m *Manager) Add(name string, cache Cache, opts ...EntryOption) error {
	if _, ok := m.M.Load(name); ok {
		return errors.New("cache entry is already exist")
	}

	entry := &Entry{
		ctx:     m.ctx,
		name:    name,
		memory:  cache,
		preload: false,
	}

	for _, opt := range opts {
		opt(entry)
	}

	go entry.run()
	m.M.Store(name, entry)
	return nil
}

func (e *Entry) run() {
	if e.fresher != nil {
		if e.preload {
			e.fresher.handle(e.memory)
		}
		if e.fresher.interval > 0 {
			ticker := time.NewTicker(e.fresher.interval)
		FRESH:
			for {
				select {
				case <-ticker.C:
					e.channel <- e.fresher.handle
				case <-e.ctx.Done():
					break FRESH
				default:
				}
			}
		}
	}

	if e.flusher != nil && e.flusher.interval > 0 {
		ticker := time.NewTicker(e.flusher.interval)
	FLUSH:
		for {
			select {
			case <-ticker.C:
				e.channel <- e.flusher.handle
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
			if ent.flusher != nil {
				ent.flusher.handle(ent.memory)
			}
			defer wait.Done()
		}(value.(*Entry))
		return true
	})
	wait.Wait()
}
