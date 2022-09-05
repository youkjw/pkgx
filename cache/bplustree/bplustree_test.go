package bplustree

import (
	"pkgx/utils"
	"testing"
)

func TestNewWith(t *testing.T) {
	tree := NewWith[int](3, utils.IntComparator[int])
	tree.Put(1, 1)
	tree.Put(2, 2)
	tree.Put(3, 3)
	tree.Put(4, 4)
	tree.Put(5, 5)
}
