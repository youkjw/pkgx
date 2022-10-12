package btree

import (
	"bytes"
	"fmt"
	"pkgx/utils"
	"strings"
	"time"
)

// Value 可以比较的value, 可以对比排序
type Value interface {
	string | utils.Number | time.Time
}

// BTree B-TREE结构
// 1. 每个节点最多有 m 个子节点
// 2. 除根节点和叶子节点，其它每个节点至少有 [m/2] （向上取整的意思）个子节点
// 3. 若根节点不是叶子节点，则其至少有2个子节点
// 4. 所有NULL节点到根节点的高度都一样
// 5. 除根节点外，其它节点都包含 n 个key，其中 [m/2] -1 <= n <= m-1
type BTree[V Value] struct {
	Root       *Node[V]            //根节点
	Comparator utils.Comparator[V] //用作对比排序
	size       int                 //存储values的数量
	m          int                 //子节点数，非叶子结点最多只有M个儿子，,最少有m/2个节点,根结点的儿子数为[2, M]
}

type Node[V Value] struct {
	Parent   *Node[V]    //父节点
	Children []*Node[V]  //子节点
	Entries  []*Entry[V] //当前节点的关键字，非叶子节点关键字至少2/3个，即块的最低使用率为2/3
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
		return false
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

// 找中间的关键字
func (tree *BTree[V]) middle() int {
	return (tree.m - 1) / 2 // 关键字原本就比子节点数减少1(m-1)
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

func (tree *BTree[V]) Remove(key V) (value any, found bool) {
	node, index, found := tree.searchRecursively(tree.Root, key)
	if found {
		tree.delete(node, index)
		tree.size--
	}
	return nil, false
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
	return (tree.m + 1) / 2 //节点数量范围 (m/2向上取整 - m)
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
		left.Children = append([]*Node[V](nil), node.Children[:middle+1]...)
		right.Children = append([]*Node[V](nil), node.Children[middle+1:]...)
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

func (tree *BTree[V]) delete(node *Node[V], index int) {
	// 删除叶子节点元素
	if tree.isLeaf(node) {
		deletedKey := node.Entries[index].Key
		tree.deleteEntry(node, index)
		tree.rebalance(node, deletedKey)
		if len(tree.Root.Entries) == 0 {
			tree.Root = nil
		}
		return
	}

	//删除非叶子节点、非根节点元素时
	leftLargestNode := tree.right(node.Children[index])                // 向最下面的叶子节点借一个节点
	leftLatgeEntryIndex := len(leftLargestNode.Entries) - 1            // 叶子节点减少一个元素
	node.Entries[index] = leftLargestNode.Entries[leftLatgeEntryIndex] // 将叶子节点的元素提上来
	deletedKey := leftLargestNode.Entries[leftLatgeEntryIndex].Key     // 删除掉叶子节点对应的元素
	tree.deleteEntry(leftLargestNode, leftLatgeEntryIndex)
	tree.rebalance(leftLargestNode, deletedKey)
}

func (tree *BTree[V]) deleteEntry(node *Node[V], index int) {
	if index >= len(node.Entries) {
		return
	}

	copy(node.Entries[index:], node.Entries[index+1:])
	node.Entries[len(node.Entries)-1] = nil
	node.Entries = node.Entries[:len(node.Entries)-1]
}

func (tree *BTree[V]) deleteChild(node *Node[V], index int) {
	if index >= len(node.Children) {
		return
	}
	copy(node.Children[index:], node.Children[index+1:])
	node.Children[len(node.Children)-1] = nil
	node.Children = node.Children[:len(node.Children)-1]
}

// 删除非叶子节点、非根节点后重新平衡btree
func (tree *BTree[V]) rebalance(node *Node[V], deletedKey V) {
	// 检查是否需要重新平衡, 当节点元素小于子节点减1需要重新平衡元素
	if node == nil || len(node.Entries) >= tree.minEntries() {
		return
	}

	// 尝试向左兄弟借
	leftSibling, leftSiblingIndex := tree.leftSibling(node, deletedKey)
	if leftSibling != nil && len(leftSibling.Entries) > tree.minEntries() {
		// rotate right
		node.Entries = append([]*Entry[V]{node.Parent.Entries[leftSiblingIndex]}, node.Entries...) // 将父节点的节点减1(子节点比关键字多1)对应的关键要到当前调整节点最左边，向左兄弟节点借比当前关键字都小
		node.Parent.Entries[leftSiblingIndex] = leftSibling.Entries[len(leftSibling.Entries)-1]    // 父节点原来位置则从左兄弟节点最后的关键字提上去
		tree.deleteEntry(leftSibling, len(leftSibling.Entries)-1)                                  // 删除掉左兄弟节点最后的关键字
		if !tree.isLeaf(leftSibling) {                                                             // 左兄弟节点非叶子节点
			leftSiblingRightMostChild := leftSibling.Children[len(leftSibling.Children)-1] // 由于左兄弟节点借走了一个关键字, 左兄弟节点原来关键字右边的子节点需要调整
			leftSiblingRightMostChild.Parent = node
			node.Children = append([]*Node[V]{leftSiblingRightMostChild}, node.Children...) // 左兄弟节点原来关键字右边的子节点直接给当前调整节点的最左边
			tree.deleteChild(leftSibling, len(leftSibling.Children)-1)                      // 然后删除左兄弟节点原来关键字右边的子节点
		}
		return
	}

	// 尝试向右兄弟借
	rightSibling, rightSiblingIndex := tree.rightSibling(node, deletedKey)
	if rightSibling != nil && len(rightSibling.Entries) > tree.minEntries() {
		// rotate left
		node.Entries = append(node.Entries, node.Parent.Entries[rightSiblingIndex-1]) // 将父节点的节点减1(子节点比关键字多1)对应的关键自要到当前调整节点最右边，向右兄弟节点借比当前关键字都大
		node.Parent.Entries[rightSiblingIndex-1] = rightSibling.Entries[0]            // 父节点原来位置则从左节点最后的关键字提上去
		tree.deleteEntry(rightSibling, 0)                                             // 删除掉右兄弟节点最前面的关键字
		if !tree.isLeaf(rightSibling) {
			rightSiblingLeftMostChild := rightSibling.Children[0]
			rightSiblingLeftMostChild.Parent = node
			node.Children = append(node.Children, rightSiblingLeftMostChild)
			tree.deleteChild(rightSibling, 0)
		}
		return
	}

	// 左右兄弟关键字都不富有(子节点大于m/2 - 1), 就合并关键字
	if rightSibling != nil {
		// 存在右兄弟节点，但右兄弟节点不富有，合并 [当前节点所有关键字]、[当前节点对应父节点位置-1的关键字]、[右节点的所有关键字]
		node.Entries = append(node.Entries, node.Parent.Entries[rightSiblingIndex-1])
		node.Entries = append(node.Entries, rightSibling.Entries...)
		deletedKey = node.Parent.Entries[rightSiblingIndex-1].Key
		tree.deleteEntry(node.Parent, rightSiblingIndex-1)                 // 删除掉当前节点对应父节点位置-1的关键字
		tree.appendChildren(node.Parent.Children[rightSiblingIndex], node) // 向右合并，将当前节点的子节点和右兄弟节点的子节点合并，
		tree.deleteChild(node.Parent, rightSiblingIndex)                   // 删除掉当前节对应父节点的右兄弟节点
	} else if leftSibling != nil {
		// merge with left sibling
		entries := append([]*Entry[V](nil), leftSibling.Entries...)
		entries = append(entries, node.Parent.Entries[leftSiblingIndex])
		node.Entries = append(entries, node.Entries...)
		deletedKey = node.Parent.Entries[leftSiblingIndex].Key
		tree.deleteEntry(node.Parent, leftSiblingIndex)
		tree.prependChildren(node.Parent.Children[leftSiblingIndex], node)
		tree.deleteChild(node.Parent, leftSiblingIndex)
	}

	// 当前调整节点的父节点是根节点并且根节点没有关键字, 则将当前节点提升为根节点
	if node.Parent == tree.Root && len(tree.Root.Entries) == 0 {
		tree.Root = node
		node.Parent = nil
		return
	}

	// 由于父节点经过调整，不确定是否仍然富有，在以父节点为调整节点做平衡
	tree.rebalance(node.Parent, deletedKey)
}

// 如果存在则返回节点的左兄弟和元素索引（在父节点中），否则返回 (nil,-1)
func (tree *BTree[V]) leftSibling(node *Node[V], key V) (*Node[V], int) {
	if node.Parent != nil {
		index, _ := tree.search(node.Parent, key)
		index--
		if index >= 0 && index < len(node.Parent.Children) {
			return node.Parent.Children[index], index
		}
	}
	return nil, -1
}

// 如果存在则返回节点的右兄弟和元素索引（在父节点中），否则返回 (nil,-1)
func (tree *BTree[V]) rightSibling(node *Node[V], key V) (*Node[V], int) {
	if node.Parent != nil {
		index, _ := tree.search(node.Parent, key)
		index++
		if index < len(node.Parent.Children) {
			return node.Parent.Children[index], index
		}
	}
	return nil, -1
}

// 获取最左子节点
func (tree *BTree[V]) left(node *Node[V]) *Node[V] {
	if tree.Empty() {
		return nil
	}
	current := node
	for {
		if tree.isLeaf(current) {
			return current
		}
		current = current.Children[0]
	}
}

// 获取最右子节点
func (tree *BTree[V]) right(node *Node[V]) *Node[V] {
	if tree.Empty() {
		return nil
	}
	current := node
	for {
		if tree.isLeaf(current) {
			return current
		}
		current = current.Children[len(current.Children)-1]
	}
}

func (tree *BTree[V]) prependChildren(fromNode *Node[V], toNode *Node[V]) {
	children := append([]*Node[V](nil), fromNode.Children...)
	toNode.Children = append(children, toNode.Children...)
	setParent(fromNode.Children, toNode)
}

func (tree *BTree[V]) appendChildren(fromNode *Node[V], toNode *Node[V]) {
	toNode.Children = append(toNode.Children, fromNode.Children...)
	setParent(fromNode.Children, toNode)
}

// String returns a string representation of container (for debugging purposes)
func (tree *BTree[V]) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("BTree\n")
	if !tree.Empty() {
		tree.output(&buffer, tree.Root, 0, true)
	}
	return buffer.String()
}

func (tree *BTree[V]) output(buffer *bytes.Buffer, node *Node[V], level int, isTail bool) {
	for e := 0; e < len(node.Entries)+1; e++ {
		if e < len(node.Children) {
			tree.output(buffer, node.Children[e], level+1, true)
		}
		if e < len(node.Entries) {
			buffer.WriteString(strings.Repeat("    ", level))
			buffer.WriteString(fmt.Sprintf("%v", node.Entries[e].Key) + "\n")
		}
	}
}

func setParent[V Value](childrens []*Node[V], parent *Node[V]) {
	for _, node := range childrens {
		node.Parent = parent
	}
}
