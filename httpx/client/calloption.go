package client

import (
	"net/http"
	"sync"
)

type CallOption interface {
	Header(header *http.Header) error
	After(reply *http.Response)
}

type EmptyCall struct{}

func (c *EmptyCall) Header(*http.Header) error { return nil }
func (c *EmptyCall) After(*http.Response)      {}

type HeaderCall struct {
	EmptyCall
	mu     sync.RWMutex
	header map[string]string
}

func (c *HeaderCall) Header(header *http.Header) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for key, value := range c.header {
		(*header).Set(key, value)
	}
	return nil
}

func (c *HeaderCall) Add(key string, value string) {
	if c.header == nil {
		c.header = make(map[string]string)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.header[key] = value
}
