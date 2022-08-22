package eprobe

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

var (
	DefaultAddress = ":3801"
)

type EprobeState int

func (state EprobeState) State() int {
	return int(state)
}

const (
	EprobeStateOK      EprobeState = http.StatusOK       //200
	EprobeStateFailure EprobeState = http.StatusNotFound //404
)

// Eprobe 服务httpAction探针
// 返回200-399状态码为正常响应, 其余为异常
type Eprobe interface {
	LivenessProbe() EprobeState  //存活性探针
	ReadinessProbe() EprobeState //就绪性探针
	StartupProbe() EprobeState   //完成启动探针
}

type ServerOption func(srv *eprobeServer)

type eprobeServer struct {
	ctx context.Context

	*http.Server                    //server
	handler          *http.ServeMux //handler
	handlerMultiplex atomic.Value   //handler复用

	address  string
	endpoint *url.URL

	wg sync.WaitGroup
}

func WithAddress(address string) ServerOption {
	return func(srv *eprobeServer) {
		srv.address = address
	}
}

func WithHandler(handler *http.ServeMux) ServerOption {
	return func(srv *eprobeServer) {
		srv.handler = handler
	}
}

func New(ctx context.Context, opts ...ServerOption) *eprobeServer {
	mux := http.NewServeMux()
	srv := &eprobeServer{
		ctx:     ctx,
		address: DefaultAddress,
		handler: mux,
	}

	for _, opt := range opts {
		opt(srv)
	}

	// handler被替换
	if srv.handler != mux {
		// 复用其他handler
		srv.handlerMultiplex.Store(true)
	} else {
		srv.endpoint = &url.URL{
			Scheme: "http",
			Host:   srv.address,
		}
		//默认server
		srv.Server = &http.Server{
			Addr:         srv.address,
			Handler:      srv.handler,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
	}

	return srv
}

func (s *eprobeServer) Detect(p Eprobe) error {
	var err error

	s.handler.HandleFunc("/_eprobe/liveness", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(p.LivenessProbe().State())
	})
	s.handler.HandleFunc("/_eprobe/readiness", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(p.ReadinessProbe().State())
	})
	s.handler.HandleFunc("/_eprobe/startup", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(p.StartupProbe().State())
	})

	if s.handlerMultiplex.Load().(bool) {
		goto FIN
	}

	s.wg.Add(2)
	go func() {
		s.wg.Done()
		err = s.ListenAndServe()
	}()
	go func() {
		s.wg.Done()
		select {
		case <-s.ctx.Done():
			err = s.Shutdown(s.ctx)
		}
	}()
	s.wg.Wait()

FIN:
	time.Sleep(100 * time.Millisecond)
	return err
}

func (s *eprobeServer) Close() error {
	if s.handlerMultiplex.Load().(bool) {
		return nil
	}
	return s.Shutdown(s.ctx)
}

// DefaultEprobe 默认探针
type DefaultEprobe struct{}

func (d *DefaultEprobe) LivenessProbe() EprobeState {
	return EprobeStateOK
}

func (d *DefaultEprobe) ReadinessProbe() EprobeState {
	return EprobeStateOK
}

func (d *DefaultEprobe) StartupProbe() EprobeState {
	return EprobeStateOK
}
