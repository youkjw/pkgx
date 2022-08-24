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
}
