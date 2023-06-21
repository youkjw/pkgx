package list

import "pkgx/utils"

const (
	defaultSize = 128

	growthFastFactor = float32(2.0)
	growthSlowFactor = float32(1.25)
	shrinkFactor     = float32(0.25)
)

type ArrayList[V comparable] struct {
	elements []V
	size     int
}

func New[V comparable]() *ArrayList[V] {
	size := defaultSize
	list := &ArrayList[V]{
		elements: make([]V, size, size),
		size:     size,
	}
	return list
}

func (list *ArrayList[V]) Add(values ...V) {
	list.growth(len(values))
	for _, value := range values {
		list.elements[list.size] = value
		list.size++
	}
}

func (list *ArrayList[V]) Get(index int) (value V, found bool) {
	var vNil V
	if !list.withinRange(index) {
		return vNil, false
	}
	return list.elements[index], true
}

func (list *ArrayList[V]) Remove(index int) {
	if !list.withinRange(index) {
		return
	}
	var vNil V
	list.elements[index] = vNil
	copy(list.elements[index:], list.elements[index+1:])
	list.size--

	list.shrink()
}

func (list *ArrayList[V]) Contains(value V) bool {
	found := false
	for index := 0; index < list.size; index++ {
		if list.elements[index] == value {
			found = true
			break
		}
	}
	return found
}

func (list *ArrayList[V]) Values() []V {
	newElements := make([]V, list.size, list.size)
	copy(newElements, list.elements[:list.size])
	return newElements
}

func (list *ArrayList[V]) IndexOf(value V) int {
	if list.size == 0 {
		return -1
	}

	for index, element := range list.elements {
		if element == value {
			return index
		}
	}

	return -1
}

func (list *ArrayList[V]) Empty() bool {
	return list.size == 0
}

func (list *ArrayList[V]) Size() int {
	return list.size
}

func (list *ArrayList[V]) Clear() {
	list.size = 0
	list.elements = make([]V, defaultSize, defaultSize)
}

func (list *ArrayList[V]) withinRange(index int) bool {
	return index >= 0 && index < list.size
}

func (list *ArrayList[V]) Sort(comparable utils.Comparator[V]) {
	if len(list.elements) < 2 {
		return
	}
	utils.Sort[V](list.elements, comparable)
}

func (list *ArrayList[V]) Swap(i, j int) bool {
	if !list.withinRange(i) || !list.withinRange(j) {
		return false
	}
	list.elements[i], list.elements[j] = list.elements[j], list.elements[i]
	return true
}

func (list *ArrayList[V]) Insert(index int, value V) {
	if !list.withinRange(index) {
		list.Add(value)
		return
	}

	list.growth(list.size + 1)
	list.size++
	copy(list.elements[index:], list.elements[index+1:])
	list.elements[index] = value
}

func (list *ArrayList[V]) Set(index int, value V) {
	if !list.withinRange(index) {
		list.Add(value)
		return
	}
	list.elements[index] = value
}

func (list *ArrayList[V]) growth(n int) {
	listCap := cap(list.elements)
	if list.size+n > listCap {
		newCap := int(growthFastFactor * float32(listCap))
		if list.size+n > 1024 {
			newCap = int(growthSlowFactor * float32(listCap))
		}
		list.resize(newCap)
	}
}

func (list *ArrayList[V]) shrink() {
	if shrinkFactor == 0.0 {
		return
	}

	listCap := cap(list.elements)
	newCap := int((1 - shrinkFactor) * float32(listCap))
	if list.size < newCap {
		list.resize(newCap)
	}
}

func (list *ArrayList[V]) resize(n int) {
	newElements := make([]V, n, n)
	copy(newElements, list.elements)
	list.elements = newElements
}
