package httpx

type methodTree struct {
	method string
	root   *node
}

type methodTrees []methodTree

type nodeType uint8

const (
	root nodeType = iota + 1
	param
	catchAll
)

type node struct {
	path      string
	indices   string
	wildChild bool
	nType     nodeType
	priority  uint32
	children  []*node // child nodes, at most 1 :param style node at the end of the array
	handlers  HandlersChain
	fullPath  string
}
