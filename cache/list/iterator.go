package list

type IteratorCallFn func(raw any)

type Iterator[V comparable] struct {
	Current V
	List    *ArrayList[V]
	index   int
}

func (i *Iterator[V]) Next(fn IteratorCallFn) bool {
	if i.index < i.List.Size() {
		i.index++
	}

	return i.List.withinRange(i.index)
}

func (i *Iterator[V]) Prev(fn IteratorCallFn) bool {
	if i.index >= 0 {
		i.index--
	}

	return i.List.withinRange(i.index)
}
