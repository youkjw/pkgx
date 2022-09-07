package middleware

import (
	"fmt"
	"pkgx/httpx"
	"time"
)

func HandleLog() httpx.HandlerFunc {
	return func(c *httpx.Context) {
		start := time.Now()
		c.Next()
		duration := time.Now().Sub(start)
		fmt.Println(duration)
	}
}
