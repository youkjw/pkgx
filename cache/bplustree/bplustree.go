package bplustree

import (
	"bytes"
	"fmt"
	"pkgx/utils"
	"strings"
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
	sync.RWMutex
}

type Node[V Value] struct {
	Parent *Node[V]
	// 非叶子节点 len(Children)=len(Key)
	// 叶子节点没有子树, len(Children)=0
	Children []*Node[V] //对应子节点
	Key      []*V       //对应关键字
	// 叶子节点关键字对应的值
	Leaf *Leaf[V]
	// 是否是叶子节点
	isLeaf bool
	sync.RWMutex
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
	tree.Lock()
	if tree.Root == nil {
		tree.Root = &Node[V]{Key: []*V{&key}, Children: []*Node[V]{}, Leaf: &Leaf[V]{Records: []*Record[V]{record}}, isLeaf: true}
		tree.size++
		tree.Unlock()
		return
	}

	if tree.insert(tree.Root, record) {
		tree.size++
	}
	tree.Unlock()
}

func (tree *BPlusTree[V]) Get(key V) (value any, found bool) {
	tree.RLock()
	defer tree.RUnlock()
	if tree.Empty() {
		return nil, false
	}

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
	tree.Lock()
	defer tree.Unlock()
	if tree.Empty() {
		return nil, false
	}

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
		if node.isLeafNote() {
			return node, index, found
		}

		var over bool
		index, found, over = tree.searchNode(node, key)
		if over {
			index--
		}
		node = node.Children[index]
	}
}

func (tree *BPlusTree[V]) insert(node *Node[V], record *Record[V]) (inserted bool) {
	if node.isLeafNote() {
		return tree.insertIntoLeaf(node, record)
	}
	return tree.insertIntoInternal(node, record)
}

func (tree *BPlusTree[V]) insertIntoLeaf(node *Node[V], record *Record[V]) bool {
	leaf := node.Leaf
	insertPosition, found := tree.searchLeaf(leaf, record.Key)
	if found {
		//update
		leaf.Records[insertPosition] = record
		return false
	}

	// 增加叶子节点的key
	node.Key = append(node.Key, nil)
	copy(node.Key[insertPosition+1:], node.Key[insertPosition:])
	node.Key[insertPosition] = record.Key

	// 写入叶子节点records
	leaf.Records = append(leaf.Records, nil)
	copy(leaf.Records[insertPosition+1:], leaf.Records[insertPosition:])
	leaf.Records[insertPosition] = record

	// 设置parent的key
	if node.Parent != nil {
		tree.setParentKeyRecursively(node.Parent, node, record.Key)
	}

	tree.split(node)
	return true
}

func (tree *BPlusTree[V]) insertIntoInternal(node *Node[V], record *Record[V]) bool {
	insertPosition, _, over := tree.searchNode(node, record.Key)
	if over {
		// 超出范围往前取最后一个
		insertPosition--
	}
	return tree.insert(node.Children[insertPosition], record)
}

func (tree *BPlusTree[V]) searchNode(node *Node[V], key *V) (index int, found bool, over bool) {
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
			return mid, true, false
		}
	}
	// 超出切片范围
	// 查找的key比当前节点key最大值还大，未找到对应索引index时会返回多一个偏移值
	if low == len(node.Key) {
		over = true
	}
	return low, false, over
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

func (tree *BPlusTree[V]) delete(node *Node[V], index int) (value any) {
	// 从叶子节点开始删除
	if node.isLeafNote() {
		// 删除key
		key := node.Key[index]
		node.deleteKey(index)

		// 删除records
		leaf := node.Leaf
		value = leaf.Records[index]
		leaf.deleteRecord(index)

		if node.Parent != nil {
			// 获取剩下key的最大值
			maxKey := getMaxKey(node.Key)
			if maxKey != nil {
				// 修改非叶子节点上的最大key
				tree.replaceParentKeyRecursively(node.Parent, node, key, maxKey)
			}
			// 重新平衡
			tree.rebalance(node, maxKey)
		}
	}
	return
}

func (tree *BPlusTree[V]) rebalance(node *Node[V], balanceKey *V) {
	// 检查是否需要重新平衡, 当节点key小于子节点需要重新平衡元素
	if node == nil || len(node.Key) >= tree.minKey() {
		return
	}

	// 尝试向左节点借
	leftSibling, leftSiblingIndex := tree.leftSibling(node, balanceKey)
	if leftSibling != nil && len(leftSibling.Key) > tree.minKey() {
		parent := node.Parent
		node.Key = append([]*V{leftSibling.Key[len(leftSibling.Key)-1]}, node.Key...) // 将父节点的节点对应的关键要到当前调整节点最左边，向左兄弟节点借比当前关键字都小
		leftSibling.deleteKey(len(leftSibling.Key) - 1)                               // 先删除掉左兄弟节点最后的关键字
		parent.Key[leftSiblingIndex] = leftSibling.Key[len(leftSibling.Key)-1]        // 父节点原来位置则从左兄弟节点最后的关键字提上去
		if !leftSibling.isLeafNote() {                                                // 左兄弟节点非叶子节点
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
		return
	}

	// 尝试向右节点借
	rightSibling, rightSiblingIndex := tree.rightSibling(node, balanceKey)
	if rightSibling != nil && len(rightSibling.Key) > tree.minKey() {
		parent := node.Parent
		node.Key = append(node.Key, rightSibling.Key[0]) // 将父节点的节点对应的关键要到当前调整节点最右边，向右兄弟节点借比当前关键字都大
		rightSibling.deleteKey(0)                        // 先删除掉左兄弟节点首个的关键字
		if !rightSibling.isLeafNote() {
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
		// 合并后调整父节点、祖先节点的关键字
		maxKey := getMaxKey(node.Key)
		if maxKey != nil {
			tree.setParentKeyRecursively(parent, node, maxKey)
		}
		return
	}

	var maxKey *V
	// 左右兄弟关键字都不富有(子节点大于m/2), 就合并关键字
	if rightSibling != nil {
		// 存在右兄弟节点，但右兄弟节点不富有，合并 [当前节点所有关键字]、[当前节点对应父节点位置-1的关键字]、[右节点的所有关键字]
		parent := node.Parent
		node.Key = append(node.Key, rightSibling.Key...)
		balanceKey = parent.Key[rightSiblingIndex-1]
		parent.deleteKey(rightSiblingIndex - 1) // 删除掉右兄弟节点对应父节点位置-1的关键字
		tree.appendChildren(rightSibling, node) // 向左合并，将当前节点的子节点和右兄弟节点的子节点合并，
		parent.deleteChild(rightSiblingIndex)   // 删除掉当前节对应父节点的右兄弟节点
		// 合并后调整父节点、祖先节点的关键字
		maxKey = getMaxKey(node.Key)
		if maxKey != nil {
			tree.replaceParentKeyRecursively(parent, node, balanceKey, maxKey)
		}

		if node.isLeafNote() {
			// 叶子节点调整leaf
			leaf := node.Leaf
			rightSiblingLeaf := rightSibling.Leaf
			leaf.Records = append(leaf.Records, rightSiblingLeaf.Records...)
			leaf.Next = rightSiblingLeaf.Next
		}
	} else if leftSibling != nil {
		parent := node.Parent

		// merge with left sibling
		node.Key = append(leftSibling.Key, node.Key...)
		balanceKey = parent.Key[leftSiblingIndex]
		parent.deleteKey(leftSiblingIndex)
		tree.prependChildren(leftSibling, node)
		parent.deleteChild(leftSiblingIndex)
		// 合并后调整父节点、祖先节点的关键字
		maxKey = getMaxKey(node.Key)
		if maxKey != nil {
			tree.replaceParentKeyRecursively(parent, node, balanceKey, maxKey)
		}

		if node.isLeafNote() {
			// 叶子节点调整leaf
			leaf := node.Leaf
			leftSiblingLeaf := leftSibling.Leaf
			leaf.Records = append(leftSiblingLeaf.Records, leaf.Records...)
			leaf.Prev = leftSiblingLeaf.Prev
		}
	}

	// 当前调整节点的父节点是根节点并且根节点没有关键字, 则将当前节点提升为根节点
	if node.Parent == tree.Root && len(tree.Root.Key) == 0 {
		tree.Root = node
		node.Parent = nil
		return
	}

	// 由于父节点经过调整，不确定是否仍然富有，在以父节点为调整节点做平衡
	tree.rebalance(node.Parent, maxKey)
}

func (tree *BPlusTree[V]) minChildren() int {
	return (tree.maxDegree + 1) / 2 //节点数量范围 (m/2向上取整 - m)
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
	if (!node.isLeafNote() && !tree.shouldSplitChild(node)) || (node.isLeafNote() && !tree.shouldSplitLeaf(node)) {
		return
	}

	if tree.Root == node {
		tree.splitRoot()
		return
	}

	tree.splitNonRoot(node)
	return
}

func (tree *BPlusTree[V]) splitRoot() {
	node := tree.Root
	middle := tree.middle()
	left := &Node[V]{}
	right := &Node[V]{}

	// 根节点是叶子节点
	if node.isLeafNote() {
		leaf := node.Leaf
		leftLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[:middle+1]...), Prev: node.Leaf.Prev}
		rightLeaf := &Leaf[V]{Records: append([]*Record[V](nil), leaf.Records[middle+1:]...), Prev: leftLeaf, Next: node.Leaf.Next}
		leftLeaf.Next = rightLeaf
		tree.appendKey(node, getRecordsMaxKey(leftLeaf.Records))
		tree.appendKey(node, getRecordsMaxKey(rightLeaf.Records))

		left.Parent = node
		left.isLeaf = true
		left.Leaf = leftLeaf
		right.Parent = node
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
	if node.isLeafNote() {
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

	insertPosition, _, over := tree.searchNode(parent, getMaxKey(left.Key))
	if over {
		insertPosition--
	}
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertPosition+1:], parent.Children[insertPosition:])
	parent.Children[insertPosition] = left
	parent.Children[insertPosition+1] = right

	tree.split(parent)
	return
}

func (tree *BPlusTree[V]) shouldSplitLeaf(node *Node[V]) bool {
	return len(node.Key) > tree.maxKey()
}

func (tree *BPlusTree[V]) shouldSplitChild(node *Node[V]) bool {
	return len(node.Children) > tree.maxChildren()
}

func (tree *BPlusTree[V]) appendKey(node *Node[V], key *V) {
	position, found, over := tree.searchNode(node, key)
	if !found {
		if over {
			position--
		}
		node.Key = append(node.Key, nil)
		copy(node.Key[position+1:], node.Key[position:])
		node.Key[position] = key
	}
}

func (tree *BPlusTree[V]) setParentKeyRecursively(parent *Node[V], node *Node[V], key *V) {
	insertPosition, found := findNodePosition(parent, node)
	if found && tree.Comparator(*parent.Key[insertPosition], *key) < 0 {
		parent.Key[insertPosition] = key
		if parent.Parent != nil {
			tree.setParentKeyRecursively(parent.Parent, parent, key)
		}
	}
}

func (tree *BPlusTree[V]) replaceParentKeyRecursively(parent *Node[V], node *Node[V], oldKey *V, newKey *V) {
	insertPosition, found := findNodePosition(parent, node)
	if found && tree.Comparator(*parent.Key[insertPosition], *oldKey) == 0 {
		parent.Key[insertPosition] = newKey
		if parent.Parent != nil {
			tree.replaceParentKeyRecursively(parent.Parent, parent, oldKey, newKey)
		}
	}
}

func (tree *BPlusTree[V]) leftSibling(node *Node[V], Key *V) (*Node[V], int) {
	if node.Parent != nil {
		index, _, _ := tree.searchNode(node.Parent, Key)
		index--
		if index >= 0 && index < len(node.Parent.Children) {
			return node.Parent.Children[index], index
		}
	}
	return nil, -1
}

func (tree *BPlusTree[V]) rightSibling(node *Node[V], Key *V) (*Node[V], int) {
	if node.Parent != nil {
		index, _, _ := tree.searchNode(node.Parent, Key)
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

// String returns a string representation of container (for debugging purposes)
func (tree *BPlusTree[V]) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("BPlusTree\n")
	if !tree.Empty() {
		tree.output(&buffer, tree.Root, 0, true)
	}
	return buffer.String()
}

func (tree *BPlusTree[V]) output(buffer *bytes.Buffer, node *Node[V], level int, isTail bool) {
	for e := 0; e < len(node.Key)+1; e++ {
		if e < len(node.Children) {
			tree.output(buffer, node.Children[e], level+1, true)
		}
		if e < len(node.Key) {
			buffer.WriteString(strings.Repeat("    ", level))
			buffer.WriteString(fmt.Sprintf("%v", *node.Key[e]) + "\n")
		}
	}
}

func (node *Node[V]) isLeafNote() bool {
	return node.isLeaf
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

func findNodePosition[V Value](parent *Node[V], node *Node[V]) (index int, found bool) {
	for sindex, snode := range parent.Children {
		if snode == node {
			return sindex, true
		}
	}
	return -1, false
}

func getMaxKey[V Value](keys []*V) *V {
	if len(keys) > 0 {
		return keys[len(keys)-1]
	}
	return nil
}

func getRecordsMaxKey[V Value](records []*Record[V]) *V {
	if len(records) > 0 {
		return records[len(records)-1].Key
	}
	return nil
}
