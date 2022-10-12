package bplustree

import (
	"pkgx/utils"
	"sync"
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
	lock   sync.RWMutex
}

type Leaf[V Value] struct {
	Records []*Record[V] // 数据记录

	Prev *Leaf[V] //前项叶子地址
	Next *Leaf[V] //后项叶子地址
	lock sync.RWMutex
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

func (tree *BPlusTree[V]) Remove(key V) (value any, found bool) {
	// 查找到node节点
	node, _, _ := tree.searchRecursively(tree.Root, &key)
	// 查找叶节点
	index, found := tree.searchLeaf(node.Leaf, &key)
	if found {
		value = tree.delete(node, index)
		tree.size--
	}
	return nil, false
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
	node.Leaf.lock.Lock()
	if found {
		//update
		node.Leaf.Records[insertPosition] = record
		node.Leaf.lock.Unlock()
		return false
	}

	// 写入叶子节点records
	leaf := node.Leaf
	leaf.Records = append(leaf.Records, nil)
	copy(leaf.Records[:insertPosition], leaf.Records[insertPosition+1:])
	leaf.Records[insertPosition] = record
	node.Leaf.lock.Unlock()

	// 增加叶子节点的key
	node.lock.Lock()
	node.Key = append(node.Key, nil)
	copy(node.Key[:insertPosition], node.Key[insertPosition+1:])
	node.Key[insertPosition] = record.Key
	node.lock.Unlock()

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
	node.lock.RLock()
	defer node.lock.RUnlock()
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
	leaf.lock.RLock()
	defer leaf.lock.RUnlock()
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

func (tree *BPlusTree[V]) delete(node *Node[V], index int) (value any) {
	// 从叶子节点开始删除
	if tree.isLeaf(node) {
		// 删除key
		node.lock.Lock()
		key := node.Key[index]
		node.deleteKey(index)
		node.lock.Unlock()
		// 删除records
		leaf := node.Leaf
		leaf.lock.Lock()
		value = leaf.Records[index]
		leaf.deleteRecord(index)
		leaf.lock.Unlock()
		// 重新平衡
		tree.rebalance(node, key)
		// 针对父节点的删除
		tree.deleteNode(node)
	}
	return
}

func (tree *BPlusTree[V]) deleteNode(node *Node[V]) {
	for node.Parent != nil {

	}
}

func (tree *BPlusTree[V]) rebalance(node *Node[V], deletedKey *V) {
	// 检查是否需要重新平衡, 当节点key小于子节点需要重新平衡元素
	if node == nil || len(node.Key) >= tree.minKey() {
		return
	}

	// 尝试向左节点借
	leftSibling, leftSiblingIndex := tree.leftSibling(node, deletedKey)
	if leftSibling != nil && len(leftSibling.Key) > tree.minKey() {
		node.Key = append([]*V{node.Parent.Key[leftSiblingIndex]}, node.Key...)     // 将父节点的节点对应的关键要到当前调整节点最左边，向左兄弟节点借比当前关键字都小
		leftSibling.deleteKey(len(leftSibling.Key) - 1)                             // 先删除掉左兄弟节点最后的关键字
		node.Parent.Key[leftSiblingIndex] = leftSibling.Key[len(leftSibling.Key)-1] // 父节点原来位置则从左兄弟节点最后的关键字提上去
		if !tree.isLeaf(leftSibling) {                                              // 左兄弟节点非叶子节点
			leftSiblingRightMostChild := leftSibling.Children[len(leftSibling.Children)-1] // 由于左兄弟节点借走了一个关键字, 左兄弟节点原来关键字右边的子节点需要调整
			leftSiblingRightMostChild.Parent = node
			node.Children = append([]*Node[V]{leftSiblingRightMostChild}, node.Children...) // 左兄弟节点原来关键字右边的子节点直接给当前调整节点的最左边
			leftSibling.deleteChild(len(leftSibling.Children) - 1)                          // 然后删除左兄弟节点原来关键字右边的子节点
		} else { // 左兄弟节点是叶子节点
			leaf := node.Leaf                                                                 // 当前调整叶子节点
			siblingLeaf := leftSibling.Leaf                                                   // 左兄弟叶子节点
			leftSiblingRightMostRecords := siblingLeaf.Records[len(siblingLeaf.Records)-1]    // 左兄弟叶子节点最后一个records
			leaf.Records = append([]*Record[V]{leftSiblingRightMostRecords}, leaf.Records...) // 左兄弟叶子节点最后一个records调整到当前叶子节点的最左边
			siblingLeaf.deleteRecord(len(siblingLeaf.Records) - 1)                            // 删除左兄弟叶子节点最后一个records
		}
	}

	// 尝试向右节点借
	rightSibling, rightSiblingIndex := tree.rightSibling(node, deletedKey)
	if rightSibling != nil && len(rightSibling.Key) > tree.minKey() {
		node.Key = append(node.Key, node.Parent.Key[rightSiblingIndex]) // 将父节点的节点对应的关键要到当前调整节点最右边，向右兄弟节点借比当前关键字都大
		rightSibling.deleteKey(0)                                       // 先删除掉左兄弟节点首个的关键字
		node.Parent.Key[rightSiblingIndex] = rightSibling.Key[0]        // 父节点原来位置则从右兄弟节点最后的关键字提上去
		if !tree.isLeaf(rightSibling) {
			rightSiblingLeftMostChild := rightSibling.Children[0]
			rightSiblingLeftMostChild.Parent = node
			node.Children = append(node.Children, rightSiblingLeftMostChild)
			rightSibling.deleteChild(0)
		} else { // 右兄弟节点是叶子节点
			leaf := node.Leaf                                                              // 当前调整叶子节点
			siblingLeaf := rightSibling.Leaf                                               // 右兄弟叶子节点
			rightSiblingLeftMostRecords := siblingLeaf.Records[len(siblingLeaf.Records)-1] // 右兄弟叶子节点最后一个records
			leaf.Records = append(leaf.Records, rightSiblingLeftMostRecords)               // 右兄弟叶子节点最后一个records调整到当前叶子节点的最右边
			siblingLeaf.deleteRecord(len(siblingLeaf.Records) - 1)                         // 删除右兄弟叶子节点最后一个records
		}
	}

	// 左右兄弟关键字都不富有(子节点大于m/2), 就合并关键字
	if rightSibling != nil {
		// 存在右兄弟节点，但右兄弟节点不富有，合并 [当前节点所有关键字]、[当前节点对应父节点位置-1的关键字]、[右节点的所有关键字]
		node.Key = append(node.Key, rightSibling.Key...)
		deletedKey = node.Parent.Key[rightSiblingIndex-1]
		node.Parent.deleteKey(rightSiblingIndex - 1)                       // 删除掉当前节点对应父节点位置-1的关键字
		tree.appendChildren(node.Parent.Children[rightSiblingIndex], node) // 向右合并，将当前节点的子节点和右兄弟节点的子节点合并，
		node.Parent.deleteChild(rightSiblingIndex)                         // 删除掉当前节对应父节点的右兄弟节点
	} else if leftSibling != nil {
		// merge with left sibling
		node.Key = append(leftSibling.Key, node.Key...)
		deletedKey = node.Parent.Key[leftSiblingIndex]
		node.Parent.deleteKey(leftSiblingIndex)
		tree.prependChildren(node.Parent.Children[leftSiblingIndex], node)
		node.Parent.deleteChild(leftSiblingIndex)
	}

	// 当前调整节点的父节点是根节点并且根节点没有关键字, 则将当前节点提升为根节点
	if node.Parent == tree.Root && len(tree.Root.Key) == 0 {
		tree.Root = node
		node.Parent = nil
		return
	}
}

func (tree *BPlusTree[V]) isLeaf(node *Node[V]) bool {
	return node.isLeaf
}

func (tree *BPlusTree[V]) minChildren() int {
	return tree.maxDegree / 2 //节点数量范围 (m/2向上取整 - m)
}

// 获取树的最大层级
func (tree *BPlusTree[V]) maxChildren() int {
	return tree.maxDegree
}

// 找中间的关键字
func (tree *BPlusTree[V]) middle() int {
	return tree.maxDegree / 2 // 关键字与节点数相同
}

func (tree *BPlusTree[V]) maxKey() int {
	return tree.maxChildren()
}

func (tree *BPlusTree[V]) minKey() int {
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

// 分裂
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
	return len(node.Key) > tree.maxKey()
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
		parent.lock.Lock()
		parent.Key[insertPosition] = key
		parent.lock.Unlock()
		if parent.Parent != nil {
			tree.setParentKeyRecursively(parent.Parent, parent, key)
		}
	}
}

func (tree *BPlusTree[V]) leftSibling(node *Node[V], Key *V) (*Node[V], int) {
	if node.Parent != nil {
		index, _ := tree.searchNode(node.Parent, Key)
		index--
		if index >= 0 && index < len(node.Parent.Children) {
			return node.Parent.Children[index], index
		}
	}
	return nil, -1
}

func (tree *BPlusTree[V]) rightSibling(node *Node[V], Key *V) (*Node[V], int) {
	if node.Parent != nil {
		index, _ := tree.searchNode(node.Parent, Key)
		index++
		if index >= 0 && index < len(node.Parent.Children) {
			return node.Parent.Children[index], index
		}
	}
	return nil, 0
}

func (tree *BPlusTree[V]) prependChildren(fromNode *Node[V], toNode *Node[V]) {
	children := append([]*Node[V](nil), fromNode.Children...)
	toNode.Children = append(children, toNode.Children...)
	setParent(fromNode.Children, toNode)
}

func (tree *BPlusTree[V]) appendChildren(fromNode *Node[V], toNode *Node[V]) {
	toNode.Children = append(toNode.Children, fromNode.Children...)
	setParent(fromNode.Children, toNode)
}

func (node *Node[V]) deleteKey(index int) {
	if index >= len(node.Key) {
		return
	}

	copy(node.Key[index:], node.Key[index+1:])
	node.Key[len(node.Key)-1] = nil
	node.Key = node.Key[:len(node.Key)-1]
}

func (node *Node[V]) deleteChild(index int) {
	if index >= len(node.Children) {
		return
	}
	copy(node.Children[index:], node.Children[index+1:])
	node.Children[len(node.Children)-1] = nil
	node.Children = node.Children[:len(node.Children)-1]
}

func (leaf *Leaf[V]) deleteRecord(index int) {
	if index >= len(leaf.Records) {
		return
	}

	copy(leaf.Records[index:], leaf.Records[index+1:])
	leaf.Records[len(leaf.Records)-1] = nil
	leaf.Records = leaf.Records[:len(leaf.Records)-1]
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
