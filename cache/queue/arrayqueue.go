package queue

import "pkgx/cache/list"

type ArrayQueue[V comparable] struct {
	list *list.ArrayList[V]
}

func New[V comparable]() *ArrayQueue[V] {
	return &ArrayQueue[V]{list: list.New[V]()}
}

func (q *ArrayQueue[V]) Enqueue(value V) {
	q.list.Add(value)
}

func (q *ArrayQueue[V]) Dequeue() (value V, found bool) {
	if q.list.Empty() {
		return nil, false
	}
	return q.list.Get(0)
}

func (q *ArrayQueue[V]) Peek() (value V, found bool) {
	return q.list.Get(0)
}

func (q *ArrayQueue[V]) Size() int {
	return q.list.Size()
}

func (q *ArrayQueue[V]) Clear() {
	q.list.Clear()
}

func (q *ArrayQueue[V]) Values() []V {
	return q.list.Values()
}

func (q *ArrayQueue[V]) withinRange(index int) bool {
	return index >= 0 && index < q.list.Size()
}
