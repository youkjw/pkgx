package httpx

import (
	"context"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"pkgx/httpx/render"
	"strings"
	"sync"
)

var (
	defaultTrustedCIDRs = []*net.IPNet{
		{ // 0.0.0.0/0 (IPv4)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0},
		},
		{ // ::/0 (IPv6)
			IP:   net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			Mask: net.IPMask{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	defaultProxyIPHeaders = []string{"X-Forwarded-For", "X-Real-IP"}
)

const abortIndex int8 = math.MaxInt8 >> 1

type (
	HandlerFunc   func(*Context)
	HandlersChain []HandlerFunc
)

type Context struct {
	Log     Logger
	Request *http.Request
	Writer  ResponseWriter

	router *Router
	// 流程控制索引
	index int8
	mu    sync.RWMutex
	// Keys is a key/value pair exclusively for the context of each request.
	Keys map[string]any

	fullPath string
	handlers HandlersChain

	// c.Request.URL.Query
	queryCache url.Values
	// c.Request.PostForm
	formCache url.Values

	// SameSite allows a server to define a cookie attribute making it impossible for
	// the browser to send this cookie along with cross-site requests.
	sameSite http.SameSite

	// path路径上param参数 eg. /:id/:name
	// GetParam("id")|GetParam("name") 获取
	params *Params
	// 同上
	// c.Request.URL.path Params
	Params Params

	// 路由查找时如有param类型时直接挑选匹配节点
	skippedNodes *[]skippedNode
}

func (c *Context) reset() {
	c.Writer = &ehttpWriter{}
	c.handlers = nil
	c.index = -1
	c.sameSite = http.SameSiteDefaultMode
	c.Keys = nil
	c.fullPath = ""
	c.Params = c.Params[:0]
	*c.params = (*c.params)[:0]
	*c.skippedNodes = (*c.skippedNodes)[:0]
}

func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}

func (c *Context) IsAborted() bool {
	return c.index >= abortIndex
}

func (c *Context) Abort() {
	c.index = abortIndex
}

func (c *Context) AbortWithStatus(code int) {
	c.Status(code)
	c.Writer.WriteHeaderNow()
	c.Abort()
}

func (c *Context) GetFullPath() string {
	return c.fullPath
}

func (c *Context) GetBody() ([]byte, error) {
	return ioutil.ReadAll(c.Request.Body)
}

func (c *Context) GetQuery(key string) (string, bool) {
	values := c.GetQueryArray(key)
	if len(values) > 0 {
		return values[0], true
	}
	return "", false
}

func (c *Context) GetQueryArray(key string) []string {
	c.initQueryCache()
	if values, ok := c.queryCache[key]; ok {
		return values
	}
	return []string{}
}

func (c *Context) initQueryCache() {
	if c.queryCache == nil {
		if c.Request != nil {
			c.queryCache = c.Request.URL.Query()
		} else {
			c.queryCache = make(url.Values)
		}
	}
}

func (c *Context) GetPostForm(key string) (string, bool) {
	values := c.GetPostFormArray(key)
	if len(values) > 0 {
		return values[0], true
	}
	return "", false
}

func (c *Context) GetPostFormArray(key string) []string {
	c.initFormCache()
	if values, ok := c.formCache[key]; ok {
		return values
	}
	return []string{}
}

func (c *Context) initFormCache() {
	if c.formCache == nil {
		if c.Request != nil {
			c.Request.ParseMultipartForm(c.router.MaxMultipartMemory)
			c.formCache = c.Request.PostForm
		} else {
			c.formCache = make(url.Values)
		}
	}
}

func (c *Context) GetParam(key string) string {
	return c.Params.ByName(key)
}

func (c *Context) Response(r render.Render) {
	_, err := c.Writer.Write(r.Parse())
	if err != nil {
		c.Log.Errorf("http responseWrite write error:%v", err)
	}
	r.WriterContentType(c.Writer)
	c.Writer.WriteHeaderNow()
}

func (c *Context) ResponseWithCode(code int) {
	c.Status(code)
	c.Writer.WriteHeaderNow()
}

func (c *Context) ResponseWithCodeMessage(code int, defaultMessage []byte) {
	c.Status(code)
	c.Header(HeaderContentType, ContentTypeTextPlain)
	if c.Writer.Written() {
		c.Writer.WriteHeaderNow()
		return
	}
	c.Response(render.RenderPlain(defaultMessage))
}

// RedirectRequest 301｜307 重定向
func (c *Context) RedirectRequest(url string) {
	req := c.Request

	code := http.StatusMovedPermanently // Permanent redirect, request with GET method
	if req.Method != http.MethodGet {
		code = http.StatusTemporaryRedirect
	}
	http.Redirect(c.Writer, req, url, code)
	c.Writer.WriteHeaderNow()
}

// Status set http status code
func (c *Context) Status(code int) {
	c.Writer.WriteHeader(code)
}

// Header set http header field
func (c *Context) Header(key, value string) {
	if value == "" {
		c.Writer.Header().Del(key)
		return
	}
	c.Writer.Header().Set(key, value)
}

// GetHeader get request header field
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Cookie get request cookie field
func (c *Context) Cookie(name string) (string, error) {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return "", err
	}
	value, _ := url.QueryUnescape(cookie.Value)
	return value, nil
}

// SetCookie set response cookie
func (c *Context) SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	if len(path) == 0 {
		path = "/"
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(value),
		Path:     path,
		Domain:   domain,
		MaxAge:   maxAge,
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: c.sameSite,
	})
}

// SetSameSite with cookie
func (c *Context) SetSameSite(samesite http.SameSite) {
	c.sameSite = samesite
}

func (c *Context) ContentType() string {
	contextType := c.Request.Header.Get(HeaderContentType)
	for i, char := range strings.TrimSpace(contextType) {
		if char == ' ' || char == ';' {
			return contextType[:i]
		}
	}
	return contextType
}

func (c *Context) IsWebsocket() bool {
	if strings.Contains(strings.ToLower(c.Request.Header.Get("Connection")), "upgrade") &&
		strings.EqualFold(c.Request.Header.Get("Upgrade"), "websocket") {
		return true
	}
	return false
}

func (c *Context) ClientIP() string {
	remoteIP := net.ParseIP(c.RemoteIP())
	if len(remoteIP) == 0 {
		return ""
	}

	for _, proxyHeader := range defaultProxyIPHeaders {
		ip, valid := validateIP(c.Request.Header.Get(proxyHeader))
		if valid {
			return ip
		}
	}
	return remoteIP.String()
}

func validateIP(ip string) (clientIp string, valid bool) {
	if len(ip) == 0 {
		return "", false
	}
	items := strings.Split(ip, ",")
	for i := range items {
		ipStr := strings.TrimSpace(items[i])
		netIP := net.ParseIP(ipStr)
		if netIP == nil {
			continue
		}
		if !isTrustedProxy(netIP) {
			return ipStr, true
		}
	}
	return "", false
}

func isTrustedProxy(ip net.IP) bool {
	for _, cidr := range defaultTrustedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (c *Context) RemoteIP() string {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr))
	if err != nil {
		return ""
	}
	return ip
}

func (c *Context) Context() context.Context {
	return c.Request.Context()
}

func (c *Context) NewContext() context.Context {
	return context.Background()
}

func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Keys == nil {
		c.Keys = make(map[string]any)
	}
	c.Keys[key] = value
}

func (c *Context) Get(key string) (value any, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists = c.Keys[key]
	return
}
