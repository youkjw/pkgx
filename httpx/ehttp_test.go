package ehttp

import (
	"context"
	"fmt"
	"gitlab.cpp32.com/backend/epkg/web/ehttp/render"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestHttpServer_ListenAndServe(t *testing.T) {
	ctx, _ := signal.NotifyContext(context.Background(), []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2}...)

	serve := New(Name("bid"), Version("v1.0.0"), Address(":8081"))
	route := serve.Handle()
	route.GET("/", func(c *Context) {
		bd := map[string]string{"a": "test"}
		c.Response(render.RenderJson(bd))
	})
	route.GET("/a", func(c *Context) {
		fmt.Println("a")
	})
	route.GET("/a/b", func(c *Context) {
		fmt.Println("a/b")
	})
	route.GET("/a/b/:id/:name", func(c *Context) {
		fmt.Println(c.GetParam("id"), c.GetParam("name"))
	})
	route.GET("/a/c", func(c *Context) {
		fmt.Println("a/c")
	})
	route.GET("/a/d", func(c *Context) {
		fmt.Println("a/d")
	})
	route.GET("/bccd", func(c *Context) {
		fmt.Println("b")
	})
	route.GET("/c/*action", func(c *Context) {
		fmt.Println(c.GetParam("action"))
	})
	serve.Run(ctx)

	<-ctx.Done()
}
