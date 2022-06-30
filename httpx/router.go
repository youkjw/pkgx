package httpx

import (
	"regexp"
	"sync"
)

type Router struct {
	IRouter
	trees       methodTrees
	pool        sync.Pool
	maxParams   uint16
	maxSections uint16

	// UseRawPath if enabled, the url.RawPath will be used to find parameters.
	UseRawPath bool

	// UnescapePathValues if true, the path value will be unescaped.
	// If UseRawPath is false (by default), the UnescapePathValues effectively is true,
	// as url.Path gonna be used, which is already unescaped.
	UnescapePathValues bool

	// RemoveExtraSlash a parameter can be parsed from the URL even with extra slashes.
	// See the PR #1817 and issue #1644
	RemoveExtraSlash bool
}

func NewRouter() *Router {
	Router := &Router{IRouter: IRouter{
		Handlers: nil,
		basePath: "/",
		root:     true,
	}}
	Router.IRouter.Router = Router
	Router.pool.New = func() any {
		return Router.allocateContext()
	}
	return Router
}

func (Router *Router) allocateContext() *Context {
	v := make(Params, 0, Router.maxParams)
	skippedNodes := make([]skippedNode, 0, Router.maxSections)
	return &Context{
		Router:              Router,
		params:              &v,
		skippedNodes:        &skippedNodes,
		ForwardedByClientIP: true,
		RemoteIPHeaders:     defaultRemoteIPHeaders,
		trustedCIDRs:        defaultTrustedCIDRs,
	}
}

func (Router *Router) addRouter(method, path string, handlers HandlersChain) {
	assert(path[0] == '/', "path must begin with '/'")
	assert(method != "", "HTTP method can not be empty")
	assert(len(handlers) > 0, "there must be at least one handler")

	debugPrintRouter(method, path, handlers)

	root := Router.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		Router.trees = append(Router.trees, methodTree{method: method, root: root})
	}
	root.addRouter(path, handlers)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount > Router.maxParams {
		Router.maxParams = paramsCount
	}

	if sectionsCount := countSections(path); sectionsCount > Router.maxSections {
		Router.maxSections = sectionsCount
	}
}

var (
	// regEnLetter matches english letters for http method name
	regEnLetter = regexp.MustCompile("^[A-Z]+$")
)

type IRouters interface {
	Use(...HandlerFunc) IRouters
	Handle(string, string, ...HandlerFunc) IRouters
}

// IRouter a prefix and an array of handlers (middleware).
type IRouter struct {
	Handlers HandlersChain
	basePath string
	Router   *Router
	root     bool
}

// Use adds middleware to the group, see example code in GitHub.
func (Router *IRouter) Use(middleware ...HandlerFunc) IRouters {
	Router.Handlers = append(Router.Handlers, middleware...)
	return Router.returnObj()
}

func (Router *IRouter) Handle(httpMethod, relativePath string, handlers ...HandlerFunc) IRouters {
	if matched := regEnLetter.MatchString(httpMethod); !matched {
		panic("http method " + httpMethod + " is not valid")
	}
	return Router.handle(httpMethod, relativePath, handlers)
}

func (Router *IRouter) handle(httpMethod, relativePath string, handlers HandlersChain) IRouters {
	absolutePath := Router.calculateAbsolutePath(relativePath)
	handlers = Router.combineHandlers(handlers)
	Router.Router.addRouter(httpMethod, absolutePath, handlers)
	return Router.returnObj()
}

func (Router *IRouter) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(Router.Handlers) + len(handlers)
	assert(finalSize < int(abortIndex), "too many handlers")
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, Router.Handlers)
	copy(mergedHandlers[len(Router.Handlers):], handlers)
	return mergedHandlers
}

func (Router *IRouter) calculateAbsolutePath(relativePath string) string {
	return joinPaths(Router.basePath, relativePath)
}

func (Router *IRouter) returnObj() IRouters {
	if Router.root {
		return Router.Router
	}
	return Router
}
