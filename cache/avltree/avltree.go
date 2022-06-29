package avltree

import (
	"fmt"
	"pkgx/utils"
	"time"
)

type Kv interface {
	string | utils.IntV | time.Time
}

type AvlTree[V Kv] struct {
	Root       *Node[V]
	Comparator utils.Comparator[V]
	size       int
}

type Node[V Kv] struct {
	Key      V
	Value    any
	Parent   *Node[V]
	Children [2]*Node[V]
	b        int8
}

func NewWith[V Kv](comparator utils.Comparator[V]) *AvlTree[V] {
	return &AvlTree[V]{Comparator: comparator}
}

func (avl *AvlTree[V]) Put(key V, value any) {
	avl.put(key, value, nil, &avl.Root)
}

func (avl *AvlTree[V]) Get(key V) (value interface{}, found bool) {
	n := avl.GetNode(key)
	if n != nil {
		return n.Value, true
	}
	return nil, false
}

func (avl *AvlTree[V]) GetNode(key V) *Node[V] {
	n := avl.Root
	for n != nil {
		cmp := avl.Comparator(key, n.Key)
		switch {
		case cmp == 0:
			return n
		case cmp < 0:
			n = n.Children[0]
		case cmp > 0:
			n = n.Children[1]
		}
	}
	return n
}

func (avl *AvlTree[V]) Remove(key V) {
	avl.remove(key, &avl.Root)
}

func (avl *AvlTree[V]) put(key V, value any, p *Node[V], r **Node[V]) bool {
	q := *r
	if q == nil {
		avl.size++
		*r = &Node[V]{
			Key:    key,
			Value:  value,
			Parent: p,
		}
		return true
	}

	c := avl.Comparator(key, q.Key)
	if c == 0 {
		q.Key = key
		q.Value = value
		return false
	}

	if c < 0 {
		c = -1
	} else {
		c = 1
	}
	a := (c + 1) / 2
	var fix bool
	fix = avl.put(key, value, q, &q.Children[a])
	if fix {
		return putFix[V](int8(c), r)
	}

	return false
}

func (avl *AvlTree[V]) remove(key V, qp **Node[V]) bool {
	q := *qp
	if q == nil {
		return false
	}

	c := avl.Comparator(key, q.Key)
	if c == 0 {
		avl.size--
		if q.Children[1] == nil {
			if q.Children[0] != nil {
				q.Children[0].Parent = q.Parent
			}
			*qp = q.Children[0]
			return true
		}
		fix := removeMin(&q.Children[1], &q.Key, &q.Value)
		if fix {
			return removeFix[V](-1, qp)
		}
		return false
	}

	if c < 0 {
		c = -1
	} else {
		c = 1
	}
	a := (c + 1) / 2
	fix := avl.remove(key, &q.Children[a])
	if fix {
		return removeFix(int8(-c), qp)
	}
	return false
}

func removeMin[V Kv](qp **Node[V], minKey *V, minVal *interface{}) bool {
	q := *qp
	if q.Children[0] == nil {
		*minKey = q.Key
		*minVal = q.Value
		if q.Children[1] != nil {
			q.Children[1].Parent = q.Parent
		}
		*qp = q.Children[1]
		return true
	}
	fix := removeMin(&q.Children[0], minKey, minVal)
	if fix {
		return removeFix(1, qp)
	}
	return false
}

func putFix[V Kv](c int8, t **Node[V]) bool {
	s := *t
	if s.b == 0 {
		s.b = c
		return true
	}

	if s.b == -c {
		s.b = 0
		return false
	}

	if s.Children[(c+1/2)].b == c {
		s = singleRotate(c, s)
	} else {
		s = doubleRotate(c, s)
	}
	*t = s
	return false
}

func removeFix[V Kv](c int8, t **Node[V]) bool {
	s := *t
	if s.b == 0 {
		s.b = c
		return false
	}

	if s.b == -c {
		s.b = 0
		return true
	}

	a := (c + 1) / 2
	if s.Children[a].b == 0 {
		s = rotate(c, s)
		s.b = -c
		*t = s
		return false
	}

	if s.Children[a].b == c {
		s = singleRotate[V](c, s)
	} else {
		s = doubleRotate[V](c, s)
	}
	*t = s
	return true
}

// 单旋转调整
func singleRotate[V Kv](c int8, s *Node[V]) *Node[V] {
	s.b = 0
	s = rotate[V](c, s)
	s.b = 0
	return s
}

// 双旋转调整
func doubleRotate[V Kv](c int8, s *Node[V]) *Node[V] {
	a := (c + 1) / 2
	r := s.Children[a]
	s.Children[a] = rotate[V](-c, s.Children[a])
	p := rotate(c, s)

	switch {
	case p.b == c:
		s.b = -c
		r.b = 0
	case p.b == -c:
		s.b = 0
		r.b = c
	default:
		s.b = 0
		r.b = 0
	}

	p.b = 0
	return p
}

// avltree旋转原理
// 调整右子树左旋, 调整左子树右旋
// 左旋：
// 1.将当前节点右字节点复制给r作为新节点
// 2.将r节点的左子节点赋值给当前节点的右子节点, 赋值后当前节点右字节点不为空，则将当前节点右字节点的parent指向当前节点
// 3.将当前节点赋值给r节点的左子节点，r节点的父节点修改当前节点的父节点，当前节点的父节点修改为r
// 右旋原理同左转, 方向相反
func rotate[V Kv](c int8, s *Node[V]) *Node[V] {
	a := (c + 1) / 2
	r := s.Children[a]
	s.Children[a] = r.Children[a^1]
	if s.Children[a] != nil {
		s.Children[a].Parent = s
	}
	r.Children[a^1] = s
	r.Parent = s.Parent
	s.Parent = r
	return r
}

// Empty returns true if tree does not contain any nodes.
func (avl *AvlTree[V]) Empty() bool {
	return avl.size == 0
}

// Size returns the number of elements stored in the tree.
func (avl *AvlTree[V]) Size() int {
	return avl.size
}

func (avl *AvlTree[V]) bottom(d int) *Node[V] {
	n := avl.Root
	if n == nil {
		return nil
	}

	for c := n.Children[d]; c != nil; c = n.Children[d] {
		n = c
	}
	return n
}

func (avl *AvlTree[V]) Floor(key V) (floor *Node[V], found bool) {
	found = false
	n := avl.Root
	for n != nil {
		c := avl.Comparator(key, n.Key)
		switch {
		case c == 0:
			return n, true
		case c < 0:
			n = n.Children[0]
		case c > 0:
			floor, found = n, true
			n = n.Children[1]
		}
	}
	if found {
		return
	}
	return nil, false
}

// Clear removes all nodes from the tree.
func (avl *AvlTree[V]) Clear() {
	avl.Root = nil
	avl.size = 0
}

// String returns a string representation of container
func (avl *AvlTree[V]) String() string {
	str := "AVLTree\n"
	if !avl.Empty() {
		output[V](avl.Root, "", true, &str)
	}
	return str
}

//// Keys returns all keys in-order
//func (avl *AvlTree[V]) Keys() []interface{} {
//	keys := make([]interface{}, t.size)
//	it := t.Iterator()
//	for i := 0; it.Next(); i++ {
//		keys[i] = it.Key()
//	}
//	return keys
//}
//
//// Values returns all values in-order based on the key.
//func (t *AvlTree[V]) Values() []interface{} {
//	values := make([]interface{}, t.size)
//	it := t.Iterator()
//	for i := 0; it.Next(); i++ {
//		values[i] = it.Value()
//	}
//	return values
//}

// Left returns the minimum element of the AVL tree
// or nil if the tree is empty.
func (t *AvlTree[V]) Left() *Node[V] {
	return t.bottom(0)
}

// Right returns the maximum element of the AVL tree
// or nil if the tree is empty.
func (t *AvlTree[V]) Right() *Node[V] {
	return t.bottom(1)
}

// Prev returns the previous element in an inorder
// walk of the AVL tree.
func (n *Node[V]) Prev() *Node[V] {
	return n.walk1(0)
}

// Next returns the next element in an inorder
// walk of the AVL tree.
func (n *Node[V]) Next() *Node[V] {
	return n.walk1(1)
}

func (n *Node[V]) walk1(a int) *Node[V] {
	if n == nil {
		return nil
	}

	if n.Children[a] != nil {
		n = n.Children[a]
		for n.Children[a^1] != nil {
			n = n.Children[a^1]
		}
		return n
	}

	p := n.Parent
	for p != nil && p.Children[a] == n {
		n = p
		p = p.Parent
	}
	return p
}

func (n *Node[V]) Size() int {
	if n == nil {
		return 0
	}
	size := 1
	if n.Children[0] != nil {
		size += n.Children[0].Size()
	}
	if n.Children[1] != nil {
		size += n.Children[1].Size()
	}
	return size
}

func (n *Node[V]) String() string {
	return fmt.Sprintf("%v:%v", n.Key, n.Value)
}

func output[V Kv](node *Node[V], prefix string, isTail bool, str *string) {
	if node.Children[1] != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "│   "
		} else {
			newPrefix += "    "
		}
		output(node.Children[1], newPrefix, false, str)
	}
	*str += prefix
	if isTail {
		*str += "└── "
	} else {
		*str += "┌── "
	}
	*str += node.String() + "\n"
	if node.Children[0] != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		output(node.Children[0], newPrefix, true, str)
	}
}
