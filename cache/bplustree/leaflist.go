package bplustree

type LeafList[V Value] struct {
	records[V]

	Prev *LeafList[V]
	Next *LeafList[V]
}

type records[V Value] []record[V]

type record[V Value] struct {
	Key   V
	Value interface{}
}
