package httpx

import (
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"net"
	"net/http"
	"path"
)

var (
	default404Body = []byte("404 page not found")
	default405Body = []byte("405 method not allowed")
)

type Handler struct {
	engine *Router
}

func (handler *Handler) isUnsafeTrustedProxies() bool {
	return handler.isTrustedProxy(net.ParseIP("0.0.0.0")) || handler.isTrustedProxy(net.ParseIP("::"))
}

// isTrustedProxy will check whether the IP address is included in the trusted list according to Engine.trustedCIDRs
func (handler *Handler) isTrustedProxy(ip net.IP) bool {
	c := handler.engine.pool.Get().(*Context)
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

func Default() *Handler {
	hanlder := New()
	InitDebug()
	return hanlder
}

func New() *Handler {
	return &Handler{engine: NewRouter()}
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (handler *Handler) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	if handler.isUnsafeTrustedProxies() {
		debugPrint("[WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.\n" +
			"Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.")
	}

	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, handler.Handler())
	return
}

func (handler *Handler) Handler() http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(handler, h2s)
}

// ServeHTTP conforms to the http.Handler interface.
func (handler *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := handler.engine.pool.Get().(*Context)
	c.Writer.reset(w)
	c.Request = req
	c.reset()

	handler.handleHTTPRequest(c)

	handler.engine.pool.Put(c)
}

func (handler *Handler) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path
	unescape := false
	if handler.engine.UseRawPath && len(c.Request.URL.RawPath) > 0 {
		rPath = c.Request.URL.RawPath
		unescape = handler.engine.UnescapePathValues
	}

	if handler.engine.RemoveExtraSlash {
		rPath = cleanPath(rPath)
	}

	// Find root of the tree for the given HTTP method
	t := handler.engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		root := t[i].root
		// Find route in tree
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

func redirectTrailingSlash(c *Context) {
	req := c.Request
	p := req.URL.Path
	if prefix := path.Clean(c.Request.Header.Get("X-Forwarded-Prefix")); prefix != "." {
		p = prefix + "/" + req.URL.Path
	}
	req.URL.Path = p + "/"
	if length := len(p); length > 1 && p[length-1] == '/' {
		req.URL.Path = p[:length-1]
	}
	redirectRequest(c)
}

func redirectRequest(c *Context) {
	req := c.Request
	rPath := req.URL.Path
	rURL := req.URL.String()

	code := http.StatusMovedPermanently // Permanent redirect, request with GET method
	if req.Method != http.MethodGet {
		code = http.StatusTemporaryRedirect
	}
	debugPrint("redirecting request %d: %s --> %s", code, rPath, rURL)
	http.Redirect(c.Writer, req, rURL, code)
	c.Writer.WriteHeaderNow()
}
