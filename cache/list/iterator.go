package list

type Iterator[V comparable] struct {
	Current V
	List    *ArrayList[V]
	index   int
}

func (i *Iterator[V]) Next() bool {
	if i.index < i.List.Size() {
		i.index++
	}

	return i.List.withinRange(i.index)
}

func (i *Iterator[V]) Prev() bool {
	if i.index >= 0 {
		i.index--
	}

	return i.List.withinRange(i.index)
}
