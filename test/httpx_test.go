package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"pkgx/httpx"
	"testing"
)

func TestHttpHandler(t *testing.T) {
	handler := httpx.Default()
	handler.Use(first(), second())
	handler.Handle("GET", "/", func(c *httpx.Context) {
		c.Writer.Write([]byte("ip:" + c.ClientIP()))
		c.Writer.Write([]byte("test"))
	})
	err := handler.Run(":8081")
	assert.Nil(t, err)
}

func first() httpx.HandlerFunc {
	return func(c *httpx.Context) {
		fmt.Println("first")
		c.Next()
		fmt.Println("three")
	}
}

func second() httpx.HandlerFunc {
	return func(c *httpx.Context) {
		fmt.Println("second")
		c.Next()
	}
}
