package bplustree

import (
	"pkgx/utils"
	"time"
)

// Value 可以比较的value, 可以对比排序
type Value interface {
	string | utils.Number | time.Time
}

type BPlusTree[V Value] struct {
	Root       *Node[V]
	Comparator utils.Comparator[V] //用作对比排序
	size       int                 //存储values的数量
	maxDegree  int                 //最大层级
}

type Node[V Value] struct {
	Parent *Node[V]
	// 非叶子节点是*Node，叶子节点是*Entry, 最后一个指针挪到了lastOrNextNode
	// 非叶子节点 len(Children)=len(Key)
	// 叶子节点没有子树, len(Children)=0
	Children []*Node[V] //对应子节点
	Key      []*V       //对应关键字
	// 叶子节点关键字对应的值
	Leaf *Leaf[V]
	// 是否是叶子节点
	isLeaf bool
}

type Leaf[V Value] struct {
	Records []*Record[V] // 数据记录

	Prev *Leaf[V] //前项叶子地址
	Next *Leaf[V] //后项叶子地址
}

type Record[V Value] struct {
	Key   *V  //关键字
	Value any //数据项
}

func NewWith[V Value](maxDegree int, comparator utils.Comparator[V]) *BPlusTree[V] {
	return &BPlusTree[V]{
		Comparator: comparator,
		maxDegree:  maxDegree,
	}
}

func (tree *BPlusTree[V]) Put(key V, value any) {
	record := &Record[V]{Key: &key, Value: value}
	if tree.Root == nil {
		tree.Root = &Node[V]{Key: []*V{&key}, Children: []*Node[V]{}, Leaf: &Leaf[V]{Records: []*Record[V]{record}}, isLeaf: true}
		tree.size++
		return
	}

	if tree.insert(tree.Root, record) {
		tree.size++
	}
}

func (tree *BPlusTree[V]) Get(key V) (value any, found bool) {
	// 查找node节点
	node, index, found := tree.searchRecursively(tree.Root, &key)
	if found {
		return node.Leaf.Records[index].Value, true
	}

	// 查找叶节点
	index, found = tree.searchLeaf(node.Leaf, &key)
	if found {
		return node.Leaf.Records[index].Value, true
	}

	return nil, false
}

func (tree *BPlusTree[V]) Range(key V, size int) (value []any) {
	return nil
}

func (tree *BPlusTree[V]) Remove(key V) {

}

func (tree *BPlusTree[V]) searchRecursively(startNode *Node[V], key *V) (node *Node[V], index int, found bool) {
	if tree.Empty() {
		return nil, -1, false
	}

	node = startNode
	for {
		if tree.isLeaf(node) {
			return node, index, found
		}

		index, found = tree.searchNode(node, key)
		if index >= len(node.Key) {
			index--
		}
		node = node.Children[index]
	}
}

func (tree *BPlusTree[V]) insert(node *Node[V], record *Record[V]) (inserted bool) {
	if tree.isLeaf(node) {
		return tree.insertIntoLeaf(node, record)
	}
	return tree.insertIntoInternal(node, record)
}

func (tree *BPlusTree[V]) insertIntoLeaf(node *Node[V], record *Record[V]) bool {
	insertPosition, found := tree.searchLeaf(node.Leaf, record.Key)
	if found {
		//update
		node.Leaf.Records[insertPosition] = record
		return false
	}

	// record
	leaf := node.Leaf
	leaf.Records = append(leaf.Records, nil)
	copy(leaf.Records[:insertPosition], leaf.Records[insertPosition+1:])
	leaf.Records[insertPosition] = record

	// 叶子节点的key
	node.Key = append(node.Key, nil)
	copy(node.Key[:insertPosition], node.Key[insertPosition+1:])
	node.Key[insertPosition] = record.Key

	// 设置parent的key
	if node.Parent != nil {
		tree.setParentKeyRecursively(node.Parent, node, record.Key)
	}

	tree.split(node)
	return true
}

func (tree *BPlusTree[V]) insertIntoInternal(node *Node[V], record *Record[V]) bool {
	insertPosition, _ := tree.searchNode(node, record.Key)
	if !tree.isLeaf(node) {
		// 非叶子节点需要往下找插入点, 插入的key比当前节点最大值还大，非叶子节点时未找到对应关键字时会返回多一个偏移值
		if insertPosition >= len(node.Key) {
			insertPosition--
		}
		tree.insert(node.Children[insertPosition], record)
	}
	return true
}

func (tree *BPlusTree[V]) searchNode(node *Node[V], key *V) (index int, found bool) {
	low, high := 0, len(node.Key)-1
	var mid int
	for low <= high {
		mid = (low + high) / 2
		compare := tree.Comparator(*key, *node.Key[mid])
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

func (tree *BPlusTree[V]) searchLeaf(leaf *Leaf[V], key *V) (index int, found bool) {
	low, high := 0, len(leaf.Records)-1
	var mid int
	for low <= high {
		mid = (low + high) / 2
		compare := tree.Comparator(*key, *leaf.Records[mid].Key)
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

func (tree *BPlusTree[V]) isLeaf(node *Node[V]) bool {
	return node.isLeaf
}

func (tree *BPlusTree[V]) minChildren() int {
	return (tree.maxDegree + 1) / 2 //节点数量范围 (m/2向上取整 - m)
}

func (tree *BPlusTree[V]) maxChildren() int {
	return tree.maxDegree
}

// 找中间的关键字
func (tree *BPlusTree[V]) middle() int {
	return (tree.maxDegree + 1) / 2
}

func (tree *BPlusTree[V]) maxLeaf() int {
	return tree.maxChildren()
}

func (tree *BPlusTree[V]) minLeaf() int {
	return tree.minChildren()
}

func (tree *BPlusTree[V]) Empty() bool {
	return tree.size == 0
}

func (tree *BPlusTree[V]) Size() int {
	return tree.size
}

func (tree *BPlusTree[V]) Clear() {
	tree.Root = nil
	tree.size = 0
}

func (tree *BPlusTree[V]) split(node *Node[V]) {
	if (!tree.isLeaf(node) && !tree.shouldSplitChild(node)) || (tree.isLeaf(node) && !tree.shouldSplitLeaf(node)) {
		return
	}

	if tree.Root == node {
		tree.splitRoot()
		return
	}

	tree.splitNonRoot(node)
}

func (tree *BPlusTree[V]) splitRoot() {
	node := tree.Root
	middle := tree.middle()
	left := &Node[V]{}
	right := &Node[V]{}

	// 根节点是叶子节点
	if tree.isLeaf(node) {
		leaf := node.Leaf
		leftLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[:middle+1]...), Prev: node.Leaf.Prev}
		rightLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[middle+1:]...), Prev: leftLeaf, Next: node.Leaf.Next}
		leftLeaf.Next = rightLeaf
		tree.appendKey(node, getRecordsMaxKey(leftLeaf.Records))
		tree.appendKey(node, getRecordsMaxKey(rightLeaf.Records))

		left.isLeaf = true
		left.Leaf = leftLeaf
		right.isLeaf = true
		right.Leaf = rightLeaf
	} else {
		children := node.Children
		left.Children = append([]*Node[V](nil), children[:middle+1]...)
		right.Children = append([]*Node[V](nil), children[middle+1:]...)
		setParent(left.Children, left)
		setParent(right.Children, right)
	}

	left.Key = append([]*V(nil), node.Key[:middle+1]...)
	right.Key = append([]*V(nil), node.Key[middle+1:]...)

	// 新root节点
	newRoot := &Node[V]{
		Children: []*Node[V]{left, right},
		Key:      []*V{getMaxKey(left.Key), getMaxKey(right.Key)},
	}

	// 将左 右2路节点父节点设置为新root节点
	left.Parent = newRoot
	right.Parent = newRoot
	tree.Root = newRoot
}

func (tree *BPlusTree[V]) splitNonRoot(node *Node[V]) {
	middle := tree.middle()
	parent := node.Parent

	left := &Node[V]{Parent: parent}
	right := &Node[V]{Parent: parent}
	// 叶子节点
	if tree.isLeaf(node) {
		leaf := node.Leaf
		leftLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[:middle+1]...), Prev: node.Leaf.Prev}
		rightLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[middle+1:]...), Prev: leftLeaf, Next: node.Leaf.Next}
		leftLeaf.Next = rightLeaf

		left.isLeaf = true
		left.Leaf = leftLeaf
		right.isLeaf = true
		right.Leaf = rightLeaf
	} else {
		children := node.Children
		left.Children = append([]*Node[V](nil), children[:middle+1]...)
		right.Children = append([]*Node[V](nil), children[middle+1:]...)
		setParent(left.Children, left)
		setParent(right.Children, right)
	}

	left.Key = append([]*V(nil), node.Key[:middle+1]...)
	right.Key = append([]*V(nil), node.Key[middle+1:]...)
	tree.appendKey(parent, getMaxKey(left.Key))
	tree.appendKey(parent, getMaxKey(right.Key))

	insertPosition, _ := tree.searchNode(parent, getMaxKey(left.Key))
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertPosition+1:], parent.Children[insertPosition:])
	parent.Children[insertPosition] = left
	parent.Children[insertPosition+1] = right

	tree.split(parent)
}

func (tree *BPlusTree[V]) shouldSplitLeaf(node *Node[V]) bool {
	return len(node.Leaf.Records) > tree.maxLeaf()
}

func (tree *BPlusTree[V]) shouldSplitChild(node *Node[V]) bool {
	return len(node.Children) > tree.maxChildren()
}

func (tree *BPlusTree[V]) appendKey(node *Node[V], key *V) {
	position, found := tree.searchNode(node, key)
	if !found {
		node.Key = append(node.Key, nil)
		copy(node.Key[position+1:], node.Key[position:])
		node.Key[position] = key
	}
}

func (tree *BPlusTree[V]) setParentKeyRecursively(parent *Node[V], node *Node[V], key *V) {
	insertPosition, found := findNodePosition(parent.Children, node)
	if found && tree.Comparator(*parent.Key[insertPosition], *key) < 0 {
		parent.Key[insertPosition] = key
		if parent.Parent != nil {
			tree.setParentKeyRecursively(parent.Parent, parent, key)
		}
	}
}

func setParent[V Value](nodes []*Node[V], parent *Node[V]) {
	for _, node := range nodes {
		node.Parent = parent
	}
}

func findNodePosition[V Value](childrens []*Node[V], node *Node[V]) (index int, found bool) {
	for sindex, snode := range childrens {
		if snode == node {
			return sindex, true
		}
	}
	return -1, false
}

func getMaxKey[V Value](keys []*V) *V {
	return keys[len(keys)-1]
}

func getRecordsMaxKey[V Value](records []*Record[V]) *V {
	return records[len(records)-1].Key
}
