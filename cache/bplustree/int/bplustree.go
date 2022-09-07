package bplustree

import (
	"fmt"
	"pkgx/utils"
)

type BPlusTree struct {
	Root       *Node
	Comparator utils.ComparatorInt8 //用作对比排序
	size       int                  //存储values的数量
	maxDegree  int                  //最大层级
}

type Node struct {
	Parent *Node
	// 非叶子节点是*Node，叶子节点是*Entry, 最后一个指针挪到了lastOrNextNode
	// 非叶子节点 len(Children)=len(Key)
	// 叶子节点没有子树, len(Children)=0
	Children []*Node //对应子节点
	Key      []*int  //对应关键字
	// 叶子节点关键字对应的值
	Leaf *Leaf
	// 是否是叶子节点
	isLeaf bool
}

type Leaf struct {
	Records []*Record // 数据记录

	Prev *Leaf //前项叶子地址
	Next *Leaf //后项叶子地址
}

type Record struct {
	Key   *int        //关键字
	Value interface{} //数据项
}

func NewWith(maxDegree int, comparator utils.ComparatorInt8) *BPlusTree {
	return &BPlusTree{
		Comparator: comparator,
		maxDegree:  maxDegree,
	}
}

func (tree *BPlusTree) Put(key int, value interface{}) {
	record := &Record{Key: &key, Value: value}
	if tree.Root == nil {
		if tree.Root != nil {
			goto NonRoot
		}
		tree.Root = &Node{Key: []*int{&key}, Children: []*Node{}, Leaf: &Leaf{Records: []*Record{record}}, isLeaf: true}
		return
	}

NonRoot:
	if tree.insert(tree.Root, record) {
		tree.size++
	}
}

func (tree *BPlusTree) Get(key int) (value interface{}, found bool) {
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

func (tree *BPlusTree) Range(key int, size int) (value []interface{}) {
	return nil
}

func (tree *BPlusTree) Remove(key int) {

}

func (tree *BPlusTree) searchRecursively(startNode *Node, key *int) (node *Node, index int, found bool) {
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

func (tree *BPlusTree) insert(node *Node, record *Record) (inserted bool) {
	if tree.isLeaf(node) {
		return tree.insertIntoLeaf(node, record)
	}
	return tree.insertIntoInternal(node, record)
}

func (tree *BPlusTree) insertIntoLeaf(node *Node, record *Record) bool {
	insertPosition, found := tree.searchLeaf(node.Leaf, record.Key)
	if found {
		//update
		node.Leaf.Records[insertPosition] = record
		tree.size++
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

func (tree *BPlusTree) insertIntoInternal(node *Node, record *Record) bool {
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

func (tree *BPlusTree) searchNode(node *Node, key *int) (index int, found bool) {
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

func (tree *BPlusTree) searchLeaf(leaf *Leaf, key *int) (index int, found bool) {
	low, high := 0, len(leaf.Records)-1
	var mid int
	for low <= high {
		mid = (low + high) / 2
		if leaf.Records[mid].Key == nil {
			a := leaf.Records[mid]
			fmt.Println(a)
			b := leaf.Records[mid].Key
			fmt.Println(b)
			panic("leaf.Records is nil")
		}
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

func (tree *BPlusTree) isLeaf(node *Node) bool {
	return node.isLeaf
}

func (tree *BPlusTree) minChildren() int {
	return (tree.maxDegree + 1) / 2 //节点数量范围 (m/2向上取整 - m)
}

func (tree *BPlusTree) maxChildren() int {
	return tree.maxDegree
}

// 找中间的关键字
func (tree *BPlusTree) middle() int {
	return (tree.maxDegree + 1) / 2
}

func (tree *BPlusTree) maxLeaf() int {
	return tree.maxChildren()
}

func (tree *BPlusTree) minLeaf() int {
	return tree.minChildren()
}

func (tree *BPlusTree) Empty() bool {
	return tree.size == 0
}

func (tree *BPlusTree) Size() int {
	return tree.size
}

func (tree *BPlusTree) Clear() {
	tree.Root = nil
	tree.size = 0
}

func (tree *BPlusTree) split(node *Node) {
	if (!tree.isLeaf(node) && !tree.shouldSplitChild(node)) || (tree.isLeaf(node) && !tree.shouldSplitLeaf(node)) {
		return
	}

	if tree.Root == node {
		tree.splitRoot()
		return
	}

	tree.splitNonRoot(node)
}

func (tree *BPlusTree) splitRoot() {
	node := tree.Root
	middle := tree.middle()
	left := &Node{}
	right := &Node{}

	// 根节点是叶子节点
	if tree.isLeaf(node) {
		leaf := node.Leaf
		leftLeaf := &Leaf{Records: append([]*Record(nil), leaf.Records[:middle+1]...), Prev: node.Leaf.Prev}
		rightLeaf := &Leaf{Records: append([]*Record(nil), leaf.Records[middle+1:]...), Prev: leftLeaf, Next: node.Leaf.Next}
		leftLeaf.Next = rightLeaf
		tree.appendKey(node, getRecordsMaxKey(leftLeaf.Records))
		tree.appendKey(node, getRecordsMaxKey(rightLeaf.Records))

		left.isLeaf = true
		left.Leaf = leftLeaf
		right.isLeaf = true
		right.Leaf = rightLeaf
	} else {
		children := node.Children
		left.Children = append([]*Node(nil), children[:middle+1]...)
		right.Children = append([]*Node(nil), children[middle+1:]...)
		setParent(left.Children, left)
		setParent(right.Children, right)
	}

	left.Key = append([]*int(nil), node.Key[:middle+1]...)
	right.Key = append([]*int(nil), node.Key[middle+1:]...)

	// 新root节点
	newRoot := &Node{
		Children: []*Node{left, right},
		Key:      []*int{getMaxKey(left.Key), getMaxKey(right.Key)},
	}

	// 将左 右2路节点父节点设置为新root节点
	left.Parent = newRoot
	right.Parent = newRoot
	tree.Root = newRoot
}

func (tree *BPlusTree) splitNonRoot(node *Node) {
	middle := tree.middle()
	parent := node.Parent

	left := &Node{Parent: parent}
	right := &Node{Parent: parent}
	// 叶子节点
	if tree.isLeaf(node) {
		leaf := node.Leaf
		leftLeaf := &Leaf{Records: append([]*Record(nil), leaf.Records[:middle+1]...), Prev: node.Leaf.Prev}
		rightLeaf := &Leaf{Records: append([]*Record(nil), leaf.Records[middle+1:]...), Prev: leftLeaf, Next: node.Leaf.Next}
		leftLeaf.Next = rightLeaf

		left.isLeaf = true
		left.Leaf = leftLeaf
		right.isLeaf = true
		right.Leaf = rightLeaf
	} else {
		children := node.Children
		left.Children = append([]*Node(nil), children[:middle+1]...)
		right.Children = append([]*Node(nil), children[middle+1:]...)
		setParent(left.Children, left)
		setParent(right.Children, right)
	}

	left.Key = append([]*int(nil), node.Key[:middle+1]...)
	right.Key = append([]*int(nil), node.Key[middle+1:]...)
	tree.appendKey(parent, getMaxKey(left.Key))
	tree.appendKey(parent, getMaxKey(right.Key))

	insertPosition, _ := tree.searchNode(parent, getMaxKey(left.Key))
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertPosition+1:], parent.Children[insertPosition:])
	parent.Children[insertPosition] = left
	parent.Children[insertPosition+1] = right

	tree.split(parent)
}

func (tree *BPlusTree) shouldSplitLeaf(node *Node) bool {
	return len(node.Leaf.Records) > tree.maxLeaf()
}

func (tree *BPlusTree) shouldSplitChild(node *Node) bool {
	return len(node.Children) > tree.maxChildren()
}

func (tree *BPlusTree) appendKey(node *Node, key *int) {
	position, found := tree.searchNode(node, key)
	if !found {
		node.Key = append(node.Key, nil)
		copy(node.Key[position+1:], node.Key[position:])
		node.Key[position] = key
	}
}

func (tree *BPlusTree) setParentKeyRecursively(parent *Node, node *Node, key *int) {
	insertPosition, found := findNodePosition(parent.Children, node)
	if found && tree.Comparator(*parent.Key[insertPosition], *key) < 0 {
		parent.Key[insertPosition] = key
		if parent.Parent != nil {
			tree.setParentKeyRecursively(parent.Parent, parent, key)
		}
	}
}

func setParent(nodes []*Node, parent *Node) {
	for _, node := range nodes {
		node.Parent = parent
	}
}

func findNodePosition(childrens []*Node, node *Node) (index int, found bool) {
	for sindex, snode := range childrens {
		if snode == node {
			return sindex, true
		}
	}
	return -1, false
}

func getMaxKey(keys []*int) *int {
	return keys[len(keys)-1]
}

func getRecordsMaxKey(records []*Record) *int {
	return records[len(records)-1].Key
}
