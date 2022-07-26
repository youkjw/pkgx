package epprof

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"sync"
)

var _epprofOnce sync.Once

type options struct {
	addr string
}

type Option func(*options)

func getDefaultOption() *options {
	return &options{
		addr: "0.0.0.0:3333",
	}
}

func WithAddress(dsn string) Option {
	return func(o *options) {
		o.addr = dsn
	}
}

func Start(ctx context.Context, opts ...Option) {
	option := getDefaultOption()
	for _, opt := range opts {
		opt(option)
	}

	_epprofOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

		go func() {
			serve := &http.Server{Addr: option.addr, Handler: mux}
			if err := serve.ListenAndServe(); err != nil {
				panic(fmt.Sprintf("pprof: listen %s: error(%v)", option.addr, err))
			}

			select {
			case <-ctx.Done():
				serve.Shutdown(ctx)
			}
		}()
	})
}
