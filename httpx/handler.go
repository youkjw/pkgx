package httpx

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
)

var (
	default404Body = []byte("404 page not found")
	default405Body = []byte("405 method not allowed")
)

type handler struct {
	*Router
}

func (handler *handler) isUnsafeTrustedProxies() bool {
	return handler.isTrustedProxy(net.ParseIP("0.0.0.0")) || handler.isTrustedProxy(net.ParseIP("::"))
}

// isTrustedProxy will check whether the IP address is included in the trusted list according to Router.trustedCIDRs
func (handler *handler) isTrustedProxy(ip net.IP) bool {
	c := handler.Router.pool.Get().(*Context)
	if c.trustedCIDRs == nil {
		return false
	}
	for _, cidr := range c.trustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func Default() *handler {
	SetDebugMode()
	handlerObj := New()
	return handlerObj
}

func New() *handler {
	return &handler{Router: NewRouter()}
}

// Run attaches the Router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, Router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (handler *handler) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, handler.handler())
	return
}

func (handler *handler) handler() http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(handler, h2s)
}

// ServeHTTP conforms to the http.Handler interface.
func (handler *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := handler.Router.pool.Get().(*Context)
	c.reset()
	c.Writer.reset(w)
	c.Request = req

	handler.handleHTTPRequest(c)

	handler.Router.pool.Put(c)
}

func (handler *handler) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path
	unescape := false
	if handler.Router.UseRawPath && len(c.Request.URL.RawPath) > 0 {
		rPath = c.Request.URL.RawPath
		unescape = handler.Router.UnescapePathValues
	}

	if handler.Router.RemoveExtraSlash {
		rPath = cleanPath(rPath)
	}

	// Find root of the tree for the given HTTP method
	t := handler.Router.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		root := t[i].root
		// Find Router in tree
		value := root.getValue(rPath, c.params, c.skippedNodes, unescape)
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

	serveError(c, http.StatusNotFound, default404Body)
}

var mimePlain = []string{"text/plain"}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.Writer.status = code
	c.Next()
	if c.Writer.Written() {
		return
	}
	if c.Writer.Status() == code {
		c.Writer.Header()["Content-Type"] = mimePlain
		_, err := c.Writer.Write(defaultMessage)
		if err != nil {
			debugPrint("cannot write message to writer during serve error: %v", err)
		}
		return
	}
	c.Writer.WriteHeaderNow()
}
