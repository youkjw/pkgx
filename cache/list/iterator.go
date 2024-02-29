package list

type DataTrait[V any] interface {
	Size() int
	Value(index int) V
	WithinRange(int) bool
}

type Iterator[V comparable] struct {
	Current V
	List    DataTrait[V]
	index   int
}

func (i *Iterator[V]) Next() bool {
	if i.index < i.List.Size() {
		i.index++
	}

	return i.List.WithinRange(i.index)
}

func (i *Iterator[V]) Prev() bool {
	if i.index >= 0 {
		i.index--
	}

	return i.List.WithinRange(i.index)
}

func (i *Iterator[V]) Index() int {
	return i.index
}

func (i *Iterator[V]) Value() V {
	return i.List.Value(i.Index())
}
