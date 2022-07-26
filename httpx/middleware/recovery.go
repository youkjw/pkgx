package middleware

import (
	"fmt"
	"gitlab.cpp32.com/backend/epkg/web/ehttp"
	"net/http"
	"runtime/debug"
)

func Recovery() ehttp.HandlerFunc {
	return func(c *ehttp.Context) {
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
