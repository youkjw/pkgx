package list

const (
	defaultSize = 128

	growthFastFactor = float32(2.0)
	growthSlowFactor = float32(1.25)
	shrinkFactor     = float32(0.25)
)

type ArrayList[V any] struct {
	elements []V
	size     int
}

func New[V]() *ArrayList[V] {
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
	if !list.withinRange(index) {
		return nil, false
	}
	return list.elements[index], true
}

func (list *ArrayList[V]) Remove(index int) {
	if !list.withinRange(index) {
		return
	}
	list.elements[index] = nil
	copy(list.elements[index:], list.elements[index+1:])
	list.size--

	list.shrink()
}

func (list *ArrayList[V]) withinRange(index int) bool {
	return index >= 0 && index < list.size
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
	newElements := make([]V, len(list.elements), n)
	copy(newElements, list.elements)
	list.elements = newElements
}
