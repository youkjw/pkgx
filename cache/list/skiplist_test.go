package list

import (
	"fmt"
	"pkgx/utils"
	"testing"
)

var maxN = 20

func TestInsert(t *testing.T) {
	list := NewSkipList[int](utils.IntComparator[int])
	for i := 0; i <= maxN; i++ {
		list.Insert(i, i)
	}

	fmt.Println(list.String())
}
