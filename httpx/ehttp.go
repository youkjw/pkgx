package httpx

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	Default404Body = []byte("404 page not found")
	DefaultAddress = ":8080"
	DefaultLogger  = NewStdLogger()
)

type HttpServer struct {
	http.Server
	handle Handler

	id           string
	name         string
	version      string
	address      string
	endpoint     *url.URL
	timeOut      time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	idleTimeout  time.Duration
	//tls certFile、keyFile
	certFile string
	keyFile  string

	middleware HandlersChain
	logger     Logger
	wg         sync.WaitGroup
}

type ServerOption func(srv *HttpServer)

func Name(name string) ServerOption {
	return func(srv *HttpServer) {
		srv.name = name
	}
}

func Version(version string) ServerOption {
	return func(srv *HttpServer) {
		srv.version = version
	}
}

func Address(addr string) ServerOption {
	return func(srv *HttpServer) {
		srv.address = addr
	}
}

func Middleware(middleware ...HandlerFunc) ServerOption {
	return func(srv *HttpServer) {
		srv.middleware = middleware
	}
}

func TlsConfig(certFile, keyFile string) ServerOption {
	return func(srv *HttpServer) {
		srv.keyFile = keyFile
		srv.certFile = certFile
	}
}

func TimeOut(timeout time.Duration) ServerOption {
	return func(srv *HttpServer) {
		srv.timeOut = timeout
	}
}

func TimeRead(timeout time.Duration) ServerOption {
	return func(srv *HttpServer) {
		srv.readTimeout = timeout
	}
}

func TimeoutWrite(timeout time.Duration) ServerOption {
	return func(srv *HttpServer) {
		srv.writeTimeout = timeout
	}
}

func IdleTimeout(timeout time.Duration) ServerOption {
	return func(srv *HttpServer) {
		srv.idleTimeout = timeout
	}
}

func HttpHandler(handler Handler) ServerOption {
	return func(srv *HttpServer) {
		srv.Handler = handler
	}
}

func WithLogger(logger Logger) ServerOption {
	return func(srv *HttpServer) {
		srv.logger = logger
	}
}

// New creates an HTTP server by options.
func New(opts ...ServerOption) *HttpServer {
	u := uuid.New()
	s := &HttpServer{
		id:           u.String(),
		name:         "",
		version:      "v0.0.0",
		address:      DefaultAddress,
		timeOut:      1 * time.Second,
		readTimeout:  3 * time.Second,
		writeTimeout: 3 * time.Second,
		idleTimeout:  7200 * time.Second,
		middleware:   make(HandlersChain, 0),
		logger:       DefaultLogger,
		handle:       httpHandler(),
	}
	for _, opt := range opts {
		opt(s)
	}

	s.handle.WithLogger(s.logger)
	s.Server = http.Server{
		Addr:         s.address,
		Handler:      s.handle,
		ReadTimeout:  s.readTimeout,
		WriteTimeout: s.writeTimeout,
		IdleTimeout:  s.IdleTimeout,
	}
	s.handle.Use(s.filter())
	s.handle.Use(s.middleware...)
	return s
}

func (srv *HttpServer) Handle() Handler {
	return srv.handle
}

func (srv *HttpServer) Run(ctx context.Context) (err error) {
	srv.wg.Add(2)
	go func() {
		srv.wg.Done()
		err = srv.Start(ctx)
	}()
	go func() {
		srv.wg.Done()
		err = srv.Stop(ctx)
	}()
	srv.wg.Wait()

	srv.logger.Warnf("[HTTP] server listening on:%s, uuid: %s, serverName:%s version:%s", srv.address, srv.id, srv.name, srv.version)
	time.Sleep(100 * time.Millisecond)
	return err
}

func (srv *HttpServer) Start(ctx context.Context) error {
	var (
		addr string
		err  error
	)
	defer func() {
		srv.logger.Warnf("[HTTP] server closed on:%s, uuid: %s, serverName:%s version:%s err:%s", srv.address, srv.id, srv.name, srv.version, err.Error())
	}()
	addr, err = ExtractEndpoint(srv.address)
	if err != nil {
		return err
	}
	srv.endpoint = &url.URL{
		Scheme: "http",
		Host:   addr,
	}

	if len(srv.certFile) > 0 && len(srv.keyFile) > 0 {
		// tls server
		err = srv.ListenAndServeTls(srv.address, srv.certFile, srv.keyFile)
	} else {
		err = srv.ListenAndServe(srv.address)
	}
	return err
}

func (srv *HttpServer) Stop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		srv.Server.Shutdown(ctx)
	}
	return ctx.Err()
}

func (srv *HttpServer) ID() string {
	return srv.id
}

func (srv *HttpServer) Name() string {
	return srv.name
}

func (srv *HttpServer) Version() string {
	return srv.version
}

func (srv *HttpServer) Endpoint() []string {
	return []string{srv.endpoint.String()}
}

func (srv *HttpServer) ListenAndServe(address string) error {
	srv.Server.Addr = address
	return srv.Server.ListenAndServe()
}

func (srv *HttpServer) ListenAndServeTls(address, certFile, keyFile string) error {
	srv.Server.Addr = address
	return srv.Server.ListenAndServeTLS(certFile, keyFile)
}

func (srv *HttpServer) filter() HandlerFunc {
	return func(c *Context) {
		var (
			ctx       context.Context
			ctxCancel context.CancelFunc
		)
		ctx = c.Request.Context()
		if srv.timeOut > 0 {
			ctx, ctxCancel = context.WithTimeout(ctx, srv.timeOut)
		} else {
			ctx, ctxCancel = context.WithCancel(ctx)
		}
		defer ctxCancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		if ctx.Err() != nil {
			c.Log.Errorf("http filter ctx error: %v", ctx.Err())
		}
	}
}

type Handler interface {
	http.Handler
	IRouters
	Logger

	// WithLogger logger
	WithLogger(logger Logger)
}

type ehttpHandler struct {
	*Router
	Logger
}

func httpHandler() Handler {
	return &ehttpHandler{
		Router: NewRouter(),
	}
}

func (h *ehttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := h.Router.pool.Get().(*Context)
	c.Request = r
	c.Log = h.Logger
	c.reset()
	c.Writer.reset(w)

	h.handleHttpRequest(c)
	h.Router.pool.Put(c)
}

func (h *ehttpHandler) handleHttpRequest(c *Context) {
	var (
		httpMethod = c.Request.Method
		rPath      = c.Request.URL.Path
	)
	if h.useRawPath && len(c.Request.URL.RawPath) > 0 {
		rPath = c.Request.URL.RawPath
	}

	t := h.trees
	for i := 0; i < len(t); i++ {
		if t[i].method != httpMethod {
			continue
		}

		root := t[i].root
		value := root.getValue(rPath, c.params, c.skippedNodes, true)
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.Writer.WriteHeaderNow()
			return
		}
		break
	}

	c.ResponseWithCodeMessage(http.StatusNotFound, Default404Body)
}

func (h *ehttpHandler) WithLogger(logger Logger) {
	h.Logger = logger
}
