package middleware

import (
	"fmt"
	"net/http"
	"pkgx/httpx"
	"runtime/debug"
)

func Recovery() httpx.HandlerFunc {
	return func(c *httpx.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())
				c.Log.Errorf(fmt.Sprintf("panic serving %s error: %v, %s", c.RemoteIP(), err, stack))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
