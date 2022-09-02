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
	m          int                 //子节点数，最多只有M个儿子,最少有m/2个节点,根结点的儿子数为[2, M]
}

type Node[V Value] struct {
	Parent *Node[V]
	// 非叶子节点是*Node，叶子节点是*Entry, 最后一个指针挪到了lastOrNextNode
	// 非叶子节点 len(Children)=len(Key)
	// 叶子节点没有子树, len(Children)=0
	Children []*Node[V] //对应子节点
	Key      []V        //对应关键字
	// 是否是叶子节点
	isLeaf bool
}

type Entry[V Value] struct {
	Key   V
	Value any
}

func NewWith[V Value](m int, comparator utils.Comparator[V]) *BPlusTree[V] {
	return &BPlusTree[V]{
		Comparator: comparator,
		m:          m,
	}
}

func (tree *BPlusTree[V]) Put(key V, value any) {
	if tree.Root == nil {
		tree.Root = &Node[V]{Key: []V{key}, Children: []*Node[V]{}, isLeaf: true}
		tree.size++
		return
	}

}

func (tree *BPlusTree[V]) insert() {

}
