package httpx

import (
	"regexp"
	"sync"
)

type Router struct {
	IRoute
	trees       methodTrees
	pool        sync.Pool
	maxParams   uint16
	maxSections uint16
}

func New() *Router {
	route := &Router{IRoute: IRoute{
		Handlers: nil,
		basePath: "/",
		root:     true,
	}}
	route.IRoute.engine = route
	route.pool.New = func() any {
		return route.allocateContext()
	}
	return route
}

func (router *Router) allocateContext() *Context {
	return &Context{
		router:              router,
		ForwardedByClientIP: true,
		RemoteIPHeaders:     defaultRemoteIPHeaders,
		trustedCIDRs:        defaultTrustedCIDRs,
	}
}

func (router *Router) addRoute(method, path string, handlers HandlersChain) {
	assert(path[0] == '/', "path must begin with '/'")
	assert(method != "", "HTTP method can not be empty")
	assert(len(handlers) > 0, "there must be at least one handler")

	debugPrintRoute(method, path, handlers)

	root := router.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		router.trees = append(router.trees, methodTree{method: method, root: root})
	}
	root.addRoute(path, handlers)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount > router.maxParams {
		router.maxParams = paramsCount
	}

	if sectionsCount := countSections(path); sectionsCount > router.maxSections {
		router.maxSections = sectionsCount
	}
}

var (
	// regEnLetter matches english letters for http method name
	regEnLetter = regexp.MustCompile("^[A-Z]+$")
)

type IRoutes interface {
	Use(...HandlerFunc) IRoutes
	Handle(string, string, ...HandlerFunc) IRoutes
}

// IRoute a prefix and an array of handlers (middleware).
type IRoute struct {
	Handlers HandlersChain
	basePath string
	engine   *Router
	root     bool
}

// Use adds middleware to the group, see example code in GitHub.
func (route *IRoute) Use(middleware ...HandlerFunc) IRoutes {
	route.Handlers = append(route.Handlers, middleware...)
	return route.returnObj()
}

func (route *IRoute) Handle(httpMethod, relativePath string, handlers ...HandlerFunc) IRoutes {
	if matched := regEnLetter.MatchString(httpMethod); !matched {
		panic("http method " + httpMethod + " is not valid")
	}
	return route.handle(httpMethod, relativePath, handlers)
}

func (route *IRoute) handle(httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	absolutePath := route.calculateAbsolutePath(relativePath)
	handlers = route.combineHandlers(handlers)
	route.engine.addRoute(httpMethod, absolutePath, handlers)
	return route.returnObj()
}

func (route *IRoute) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(route.Handlers) + len(handlers)
	assert(finalSize < int(abortIndex), "too many handlers")
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, route.Handlers)
	copy(mergedHandlers[len(route.Handlers):], handlers)
	return mergedHandlers
}

func (route *IRoute) calculateAbsolutePath(relativePath string) string {
	return joinPaths(route.basePath, relativePath)
}

func (route *IRoute) returnObj() IRoutes {
	if route.root {
		return route.engine
	}
	return route
}
