package ehttp

import (
	"net/http"
	"regexp"
	"sync"
)

var (
	// regEnLetter matches letters for http method name
	regEnLetter = regexp.MustCompile("^[A-Z]+$")
	// default Tree Size
	defaultTreeSize = 32
	// defaultMultipartMemory size
	defaultMultipartMemory int64 = 32 << 20 //32MB
)

type Router struct {
	IRouter
	trees methodTrees
	pool  sync.Pool

	// useRawPath if enabled, the url.RawPath will be used to find parameters.
	useRawPath bool
	// MaxMultipartMemory value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	MaxMultipartMemory int64
	// 记录所有路由中路径参数最多的数量
	maxParams uint8
	// 记录所有路由中路径节点最多的数量
	maxSections uint8
}

func NewRouter() *Router {
	r := &Router{
		IRouter: IRouter{
			Router:   nil,
			basePath: "/",
			Handlers: make(HandlersChain, 0, 8),
		},
		trees:              make(methodTrees, 0, defaultTreeSize),
		MaxMultipartMemory: defaultMultipartMemory,
	}
	r.Router = r
	r.pool.New = func() any {
		return r.newContext()
	}
	return r
}

func (Router *Router) newContext() *Context {
	params := make(Params, 0, Router.maxParams)
	sections := make([]skippedNode, 0, Router.maxSections)
	return &Context{
		router:       Router,
		params:       &params,
		skippedNodes: &sections,
	}
}

func (Router *Router) addRouter(method, path string, handlers HandlersChain) {
	root := Router.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		Router.trees = append(Router.trees, methodTree{method: method, root: root})
	}
	root.addRouter(path, handlers)

	// Update maxParams
	if paramsCount := uint8(countParams(path)); paramsCount > Router.maxParams {
		Router.maxParams = paramsCount
	}

	if sectionsCount := uint8(countSections(path)); sectionsCount > Router.maxSections {
		Router.maxSections = sectionsCount
	}
}

func (Router *Router) SetUseRawPath(b bool) {
	Router.useRawPath = b
}

func (Router *Router) SetMaxMultipartMemory(size int64) {
	Router.MaxMultipartMemory = size
}

type IRouters interface {
	Use(...HandlerFunc) IRouters
	Handle(string, string, ...HandlerFunc) IRouters
	GET(string, ...HandlerFunc) IRouters
	POST(string, ...HandlerFunc) IRouters
	DELETE(string, ...HandlerFunc) IRouters
	PUT(string, ...HandlerFunc) IRouters
	OPTIONS(string, ...HandlerFunc) IRouters
	HEAD(string, ...HandlerFunc) IRouters
}

type IRouter struct {
	Router   *Router
	basePath string
	Handlers HandlersChain
}

func (Router *IRouter) BasePath() string {
	return Router.basePath
}

func (Router *IRouter) Use(middleware ...HandlerFunc) IRouters {
	size := len(Router.Handlers) + len(middleware)
	if size > int(abortIndex) {
		panic("http handlers exceed the limit")
		return Router
	}
	Router.Handlers = append(Router.Handlers, middleware...)
	return Router
}

func (Router *IRouter) Handle(httpMethod, relativePath string, handlers ...HandlerFunc) IRouters {
	if matched := regEnLetter.MatchString(httpMethod); !matched {
		panic("http method " + httpMethod + " is not valid")
	}
	return Router.handle(httpMethod, relativePath, handlers)
}

func (Router *IRouter) handle(httpMethod, relativePath string, handlers HandlersChain) IRouters {
	absolutePath := Router.generateAbsolutePath(relativePath)
	handlers = Router.combineHandlers(handlers)
	Router.Router.addRouter(httpMethod, absolutePath, handlers)
	return Router
}

func (Router *IRouter) combineHandlers(handlers HandlersChain) HandlersChain {
	size := len(Router.Handlers) + len(handlers)
	if size > int(abortIndex) {
		panic("http handlers exceed the limit")
	}
	mergerHandlers := make(HandlersChain, size)
	copy(mergerHandlers, Router.Handlers)
	copy(mergerHandlers[len(Router.Handlers):], handlers)
	return mergerHandlers
}

func (Router *IRouter) generateAbsolutePath(relativePath string) string {
	return joinPath(Router.basePath, relativePath)
}

func (Router *IRouter) POST(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.Handle("POST", relativePath, handlers...)
}

func (Router *IRouter) GET(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.Handle("GET", relativePath, handlers...)
}

func (Router *IRouter) DELETE(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.Handle("DELETE", relativePath, handlers...)
}

// PUT is a shortcut for router.Handle("PUT", path, handle).
func (Router *IRouter) PUT(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.handle(http.MethodPut, relativePath, handlers)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handle).
func (Router *IRouter) OPTIONS(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.handle(http.MethodOptions, relativePath, handlers)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handle).
func (Router *IRouter) HEAD(relativePath string, handlers ...HandlerFunc) IRouters {
	return Router.handle(http.MethodHead, relativePath, handlers)
}
