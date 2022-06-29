package avltree

import (
	"pkgx/utils"
	"testing"
)

func TestAvlTree_Put(t *testing.T) {
	tree := NewWith[int](utils.IntComparator[int])
	tree.Put(1, "a")
	tree.Put(2, "B")
	tree.Put(3, "V")
	tree.Put(4, "c")
	tree.Put(5, "d")
	tree.Put(5, "da")

	t.Log(tree.String())
}
