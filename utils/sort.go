package utils

import "sort"

func Sort[V comparable](values []V, comparator Comparator[V]) {
	sort.Sort(sortable[V]{values: values, comparator: comparator})
}

type sortable[V comparable] struct {
	values     []V
	comparator Comparator[V]
}

func (s sortable[V]) Len() int {
	return len(s.values)
}

func (s sortable[V]) Swap(i, j int) {
	s.values[i], s.values[j] = s.values[j], s.values[i]
}

func (s sortable[V]) Less(i, j int) bool {
	return s.comparator(i, j) < 0
}
