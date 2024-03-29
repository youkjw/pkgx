package btree

import (
	"pkgx/utils"
	"testing"
)

func TestBTree_Put(t *testing.T) {
	tree := NewWith[int32](3, utils.IntComparator[int32])

	tree.Put(0, 0)
	tree.Put(1, 1)
	tree.Put(2, 2)
	tree.Put(3, 3)
	tree.Put(4, 4)
	tree.Put(5, 5)
	tree.Put(6, 6)

	tree.Put(7, 7)
	tree.Put(8, 8)
	tree.Put(9, 9)
	tree.Put(10, 10)
	tree.Put(11, 11)
	tree.Put(12, 12)
	tree.Put(13, 13)
	tree.Put(14, 14)
	tree.Put(15, 15)
	tree.Put(16, 16)
	tree.Put(17, 17)
	tree.Put(18, 18)
	tree.Put(19, 19)
	tree.Put(20, 20)
	tree.Put(21, 21)

	t.Log(tree.String())

	tree.Remove(20)
	t.Log(tree.String())
	tree.Remove(11)
	t.Log(tree.String())
	tree.Remove(12)
	t.Log(tree.String())
	tree.Remove(6)
	t.Log(tree.String())
	tree.Remove(9)
	t.Log(tree.String())
	tree.Remove(8)
	t.Log(tree.String())
	tree.Remove(0)
	t.Log(tree.String())
}

func BenchmarkPut(b *testing.B) {
	//mux := sync.Mutex{}
	tree := NewWith[int](3, utils.IntComparator[int])
	//tree.Put(1, 1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			//mux.Lock()
			tree.Put(1, 1)
			//mux.Unlock()
		}
	})
}
