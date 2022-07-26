package middleware

import (
	"fmt"
	"gitlab.cpp32.com/backend/epkg/base/elog"
	"gitlab.cpp32.com/backend/epkg/web/ehttp"
	"time"
)

func HandleLog() ehttp.HandlerFunc {
	return func(c *ehttp.Context) {
		start := time.Now()
		c.Next()
		duration := time.Now().Sub(start)

		elog.WithFields(elog.Fields{
			"url":      c.GetFullPath(),
			"ip":       c.ClientIP(),
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
			"status":   c.Writer.Status(),
			"duration": fmt.Sprintf("%4v", duration),
		}).Info("http request")
	}
}
