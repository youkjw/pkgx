package int

import (
	"fmt"
	"log"
	"pkgx/utils"
	"sync"
)

type BPlusTree struct {
	Root       *Node
	Comparator utils.ComparatorInt8 //用作对比排序
	size       int                  //存储values的数量
	maxDegree  int                  //最大层级
	mux        sync.Mutex
}

type Node struct {
	Parent *Node
	// 非叶子节点是*Node，叶子节点是*Entry, 最后一个指针挪到了lastOrNextNode
	// 非叶子节点 len(Children)=len(Key)
	// 叶子节点没有子树, len(Children)=0
	Children []*Node //对应子节点
	Key      []int   //对应关键字
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
	Key   int         //关键字
	Value interface{} //数据项
}

func NewWith(maxDegree int, comparator utils.ComparatorInt8) *BPlusTree {
	return &BPlusTree{
		Comparator: comparator,
		maxDegree:  maxDegree,
	}
}

func (tree *BPlusTree) Put(key int, value interface{}) {
	record := &Record{Key: key, Value: value}
	if tree.Root == nil {
		tree.mux.Lock()
		defer tree.mux.Unlock()
		if tree.Root != nil {
			goto NonRoot
		}
		leaf := &Leaf{Records: []*Record{record}}
		tree.Root = &Node{Key: []int{key}, Children: []*Node{}, Leaf: leaf, isLeaf: true}
		return
	}

NonRoot:
	if tree.insertIntoLeaf(tree.Root, record) {
		tree.size++
	}
}

func (tree *BPlusTree) insert(node *Node, record *Record) (inserted bool) {
	if tree.isLeaf(node) {
		return tree.insertIntoLeaf(node, record)
	}
	return true
}

func (tree *BPlusTree) insertIntoLeaf(node *Node, record *Record) bool {
	insertPosition, found := tree.searchLeaf(node.Leaf, record.Key)
	if found {
		//update
		//tree.mux.Lock()
		node.Leaf.Records[insertPosition] = record
		//tree.mux.Unlock()
		tree.size++
		return false
	}

	return true
}

func (tree *BPlusTree) searchLeaf(leaf *Leaf, key int) (index int, found bool) {
	low, high := 0, len(leaf.Records)-1
	var mid int
	for low <= high {
		mid = (low + high) / 2
		//fmt.Println(1)
		d := leaf.Records[mid]
		b := leaf.Records[mid].Key
		//fmt.Sprintf("%v", &leaf.Records[mid].Key)
		e := (*leaf.Records[mid]).Key
		g := d.Key
		v := d.Value
		//fmt.Sprintf("%v", &g)
		h := leaf.Records[mid]
		//fmt.Sprintf("%v", &h.Key)
		//fmt.Println(1)
		//time.Sleep(1)
		if b == 0 {
			log.Printf("value ---  %v", v)

			log.Printf("after --- %v", &leaf.Records[mid].Key)
			log.Printf("d --- %v", &d.Key)
			log.Printf("h --- %v", &h.Key)
			a := leaf.Records[mid]
			j := leaf.Records[mid].Key
			i := leaf.Records[mid]
			fmt.Println(a)
			fmt.Println(d)
			fmt.Println(e)
			fmt.Println(g)
			fmt.Println(h)
			c := leaf.Records[mid].Key
			fmt.Println(c)
			fmt.Println(i)
			fmt.Println(j)
			panic("leaf.Records is nil")
		}
		compare := tree.Comparator(key, leaf.Records[mid].Key)
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
