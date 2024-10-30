package list

import (
	"fmt"
	"math/bits"
	"math/rand"
	"pkgx/utils"
)

const (
	maxLevel = 25
)

type SkipList[V comparable] struct {
	start [maxLevel]*SkipListElement[V]
	end   [maxLevel]*SkipListElement[V]

	comparator  utils.Comparator[V]
	curMaxLevel int
	total       int64
}

type SkipListElement[V comparable] struct {
	level int
	key   V
	value any
	prev  *SkipListElement[V]
	next  [maxLevel]*SkipListElement[V]
}

// NewSkipList SkipList
func NewSkipList[V comparable](comparator utils.Comparator[V]) *SkipList[V] {
	return &SkipList[V]{
		start:      [maxLevel]*SkipListElement[V]{},
		end:        [maxLevel]*SkipListElement[V]{},
		comparator: comparator,
	}
}

func (list *SkipList[V]) Insert(key V, value any) {
	if list == nil {
		return
	}

	level := list.generateLevel(maxLevel)
	// 按1递增层级
	if level > list.curMaxLevel {
		level = list.curMaxLevel + 1
		list.curMaxLevel = level
	}

	elem := &SkipListElement[V]{
		level: level,
		key:   key,
		value: value,
		next:  [maxLevel]*SkipListElement[V]{},
	}

	newFirst := true
	newLast := true
	if !list.Empty() {
		newFirst = list.isMin(elem.key, list.start[0].key)
		newLast = list.isMax(elem.key, list.end[0].key)
	}

	var insertMiddle = false
	if !newFirst && !newLast {
		insertMiddle = true

		index := list.findEntryIndex(key, level)
		var currentNode, nextNode *SkipListElement[V]

		for {
			if currentNode == nil {
				nextNode = list.start[index]
			} else {
				nextNode = currentNode.next[index]
			}

			// search position
			if index <= level && (nextNode == nil || list.isMin(elem.key, nextNode.key)) {
				// 记录每一层的下一级
				// 将elem插入到currentNode和nextNode中间
				elem.next[index] = nextNode
				if currentNode != nil {
					currentNode.next[index] = elem
				}

				// 由于上级只记录第0层的, 所以只有index==0时才记录
				// 数据一定会查到到第0层
				if index == 0 {
					// 将elem插入到currentNode和nextNode中间
					elem.prev = currentNode
					if nextNode != nil {
						nextNode.prev = elem
					}
				}
			}

			if nextNode != nil && list.isMax(elem.key, nextNode.key) {
				// 继续同层级往右找
				currentNode = nextNode
			} else {
				// 往下层级找
				index--
				if index < 0 {
					break
				}
			}
		}
	}

	for i := level; i >= 0; i-- {
		if newFirst || insertMiddle {
			// 对比层级中的首个
			if list.start[i] == nil || list.isMin(elem.key, list.start[i].key) {
				if i == 0 && list.start[i] != nil {
					list.start[i].prev = elem
				}
				elem.next[i] = list.start[i]
				list.start[i] = elem
			}

			// 该层级中
			if elem.next[i] == nil {
				list.end[i] = elem
			}
		}

		if newLast {
			// 对比层级中最后一个
			if !newFirst {
				// 非首个elem
				// 将原来的的最后一个数据的next的对应层数的数据更新为elem
				// 直接将最后一个数据更新为当前elem
				if list.end[i] != nil {
					list.end[i].next[i] = elem
				}
				if i == 0 {
					elem.prev = list.end[i]
				}
				list.end[i] = elem
			}

			// 当前层级首个数据不存在
			if list.start[i] == nil || list.isMin(elem.key, list.start[i].key) {
				list.start[i] = elem
			}
		}
	}
}

func (list *SkipList[V]) Find(key V) (value any, ok bool) {
	if list == nil {
		return
	}

	return list.findRecursion(key)
}

func (list *SkipList[V]) findRecursion(key V) (value any, ok bool) {
	if list == nil || list.Empty() {
		return
	}

	//index := list.findEntryIndex(key, 0)
	//var currentNode = list.start[index]

	//if currentNode.key

	return
}

func (list *SkipList[V]) Delete() {

}

func (list *SkipList[V]) Empty() bool {
	return list.start[0] == nil
}

func (list *SkipList[V]) isEqual(key, nodeKey V) bool {
	return list.comparator(key, nodeKey) == 0
}

func (list *SkipList[V]) isMin(key, nodeKey V) bool {
	return list.comparator(key, nodeKey) < 0
}

func (list *SkipList[V]) isMax(key, nodeKey V) bool {
	return list.comparator(key, nodeKey) > 0
}

func (list *SkipList[V]) generateLevel(maxLevel int) int {
	level := maxLevel - 1
	var x = rand.Uint64() & ((1 << uint(level-1)) - 1)
	zeroCount := bits.TrailingZeros64(x)
	if zeroCount <= maxLevel {
		level = zeroCount
	}
	return level
}

func (list *SkipList[V]) findEntryIndex(key V, level int) int {
	for i := list.curMaxLevel; i >= 0; i-- {
		if list.start[i] != nil && list.isMax(key, list.start[i].key) || i <= level {
			return i
		}
	}
	return 0
}

func (list *SkipList[V]) String() string {
	s := ""

	s += " --> "
	for i, l := range list.start {
		if l == nil {
			break
		}
		if i > 0 {
			s += " -> "
		}
		next := "---"
		if l != nil {
			next = fmt.Sprintf("%v", l.key)
		}
		s += fmt.Sprintf("[%v]", next)

		if i == 0 {
			s += "    "
		}
	}
	s += "\n"

	node := list.start[0]
	for node != nil {
		s += fmt.Sprintf("%v: ", node.value)
		for i := 0; i <= node.level; i++ {

			l := node.next[i]

			next := "---"
			if l != nil {
				next = fmt.Sprintf("%v", l.key)
			}

			if i == 0 {
				prev := "---"
				if node.prev != nil {
					prev = fmt.Sprintf("%v", node.prev.key)
				}
				s += fmt.Sprintf("[%v|%v]", prev, next)
			} else {
				s += fmt.Sprintf("[%v]", next)
			}
			if i < node.level {
				s += " -> "
			}

		}
		s += "\n"
		node = node.next[0]
	}

	s += " --> "
	for i, l := range list.end {
		if l == nil {
			break
		}
		if i > 0 {
			s += " -> "
		}
		next := "---"
		if l != nil {
			next = fmt.Sprintf("%v", l.key)
		}
		s += fmt.Sprintf("[%v]", next)
		if i == 0 {
			s += "    "
		}
	}
	s += "\n"
	return s
}
