package btree

import (
	"pkgx/utils"
	"time"
)

// Value 可以比较的value, 可以对比排序
type Value interface {
	string | utils.Number | time.Time
}

// BTree B-TREE结构
type BTree[V Value] struct {
	Root       *Node[V]            //根节点
	Comparator utils.Comparator[V] //用作对比排序
	size       int                 //存储values的数量
	m          int                 //路数
}

type Node[V Value] struct {
	Parent   *Node[V]    //父节点
	Children []*Node[V]  //子节点
	Entries  []*Entry[V] //当前节点的关键字
}

type Entry[V Value] struct {
	Key   V
	Value any
}

func NewWith[V Value](m int, comparator utils.Comparator[V]) *BTree[V] {
	return &BTree[V]{
		Comparator: comparator,
		m:          m,
	}
}

func (tree *BTree[V]) Put(key V, value any) {
	entry := &Entry[V]{Key: key, Value: value}
	if tree.Root == nil {
		tree.Root = &Node[V]{Entries: []*Entry[V]{entry}, Children: []*Node[V]{}}
		tree.size++
		return
	}

	if tree.insert(tree.Root, entry) {
		tree.size++
	}
}

func (tree *BTree[V]) insert(node *Node[V], entry *Entry[V]) bool {
	if tree.isLeaf(node) {
		return tree.insertIntoLeaf(node, entry)
	}
	return tree.insertIntoInternal(node, entry)
}

func (tree *BTree[V]) insertIntoLeaf(node *Node[V], entry *Entry[V]) bool {
	insertPositon, found := tree.search(node, entry.Key)
	if found {
		node.Entries[insertPositon] = entry
	}
	node.Entries = append(node.Entries, nil)
	copy(node.Entries[:insertPositon], node.Entries[insertPositon+1:])
	node.Entries[insertPositon] = entry
	tree.split(node)
	return true
}

func (tree *BTree[V]) insertIntoInternal(node *Node[V], entry *Entry[V]) bool {
	insertPosition, found := tree.search(node, entry.Key)
	if found {
		node.Entries[insertPosition] = entry
		return false
	}
	return tree.insert(node.Children[insertPosition], entry)
}

func (tree *BTree[V]) isLeaf(node *Node[V]) bool {
	return len(node.Children) == 0
}

func (tree *BTree[V]) isFull(node *Node[V]) bool {
	return len(node.Entries) == tree.maxEntries()
}

func (tree *BTree[V]) middle() int {
	return (tree.m - 1) / 2
}

func (tree *BTree[V]) search(node *Node[V], key V) (index int, found bool) {
	low, high := 0, len(node.Entries)-1
	var mid int
	for low <= high {
		mid = (low + high) / 2
		compare := tree.Comparator(key, node.Entries[mid].Key)
		switch {
		case compare > 0:
			low = mid + 1
		case compare < 0:
			high = mid - 1
		case compare == 0:
			return mid, true
		}
	}

	return low, false
}

func (tree *BTree[V]) Get(key V) (value any, found bool) {
	node, index, found := tree.searchRecursively(tree.Root, key)
	if found {
		return node.Entries[index].Value, true
	}
	return nil, false
}

func (tree *BTree[V]) searchRecursively(startNode *Node[V], key V) (node *Node[V], index int, found bool) {
	if tree.Empty() {
		return nil, -1, false
	}

	node = startNode
	for {
		index, found = tree.search(node, key)
		if found {
			return node, index, true
		}

		if tree.isLeaf(node) {
			return nil, -1, false
		}
		node = node.Children[index]
	}
}

func (tree *BTree[V]) Remove() {

}

func (tree *BTree[V]) GetNode(key V) *Node[V] {
	node, _, _ := tree.searchRecursively(tree.Root, key)
	return node
}

func (tree *BTree[V]) Empty() bool {
	return tree.size == 0
}

func (tree *BTree[V]) Size() int {
	return tree.size
}

func (tree *BTree[V]) Clear() {
	tree.Root = nil
	tree.size = 0
}

func (tree *BTree[V]) split(node *Node[V]) {
	if !tree.shouldSplit(node) {
		return
	}

	if tree.Root == node {
		tree.splitRoot()
		return
	}

	tree.splitNonRoot(node)
}

func (tree *BTree[V]) shouldSplit(node *Node[V]) bool {
	return len(node.Entries) > tree.maxEntries()
}

func (tree *BTree[V]) maxEntries() int {
	return tree.maxChildren() - 1
}

func (tree *BTree[V]) minEntries() int {
	return tree.minChildren() - 1
}

func (tree *BTree[V]) maxChildren() int {
	return tree.m
}

func (tree *BTree[V]) minChildren() int {
	return (tree.m + 1) / 2
}

func (tree *BTree[V]) splitRoot() {
	middle := tree.middle()
	left := &Node[V]{Entries: append([]*Entry[V](nil), tree.Root.Entries[:middle]...)}
	right := &Node[V]{Entries: append([]*Entry[V](nil), tree.Root.Entries[middle+1:]...)}

	// 根节点非叶节点
	if !tree.isLeaf(tree.Root) {
		left.Children = append([]*Node[V](nil), tree.Root.Children[:middle+1]...)
		right.Children = append([]*Node[V](nil), tree.Root.Children[middle+1:]...)
		setParent[V](left.Children, left)
		setParent[V](right.Children, right)
	}

	// 新root节点
	newRoot := &Node[V]{
		Entries:  []*Entry[V]{tree.Root.Entries[middle]},
		Children: []*Node[V]{left, right},
	}

	// 将左 右2路节点父节点设置为新root节点
	left.Parent = newRoot
	right.Parent = newRoot
	tree.Root = newRoot
}

func (tree *BTree[V]) splitNonRoot(node *Node[V]) {
	middle := tree.middle()
	parent := node.Parent

	left := &Node[V]{Entries: append([]*Entry[V](nil), node.Entries[:middle]...), Parent: parent}
	right := &Node[V]{Entries: append([]*Entry[V](nil), node.Entries[middle+1:]...), Parent: parent}

	// 非叶节点
	if !tree.isLeaf(node) {
		left.Children = append([]*Node[V](nil), tree.Root.Children[:middle+1]...)
		right.Children = append([]*Node[V](nil), tree.Root.Children[middle+1:]...)
		setParent[V](left.Children, left)
		setParent[V](right.Children, right)
	}

	insertPosition, _ := tree.search(parent, node.Entries[middle].Key)

	parent.Entries = append(parent.Entries, nil)
	copy(parent.Entries[insertPosition+1:], parent.Entries[insertPosition:])
	parent.Entries[insertPosition] = node.Entries[middle]

	parent.Children[insertPosition] = left

	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertPosition+2:], parent.Children[insertPosition+1:])
	parent.Children[insertPosition+1] = right

	tree.split(parent)
}

func setParent[V Value](childrens []*Node[V], parent *Node[V]) {
	for _, node := range childrens {
		node.Parent = parent
	}
}
