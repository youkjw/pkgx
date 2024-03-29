package bplustree

import (
	"fmt"
	"pkgx/utils"
	"sync"
	"testing"
)

func TestNewWith(t *testing.T) {
	tree := NewWith[int](3, utils.IntComparator[int])
	//tree.Put(1, 1)
	//tree.Put(2, 2)
	//tree.Put(3, 3)
	//tree.Put(4, 4)
	//tree.Put(5, 5)
	//tree.Put(6, 6)
	//tree.Put(7, 7)
	//tree.Put(8, 8)
	//tree.Put(9, 9)
	//tree.Put(10, 10)
	//tree.Put(11, 11)
	//tree.Put(12, 12)
	//tree.Put(13, 13)
	//tree.Put(14, 14)
	//tree.Put(15, 15)
	//tree.Put(16, 16)
	//tree.Put(17, 17)
	//tree.Put(18, 18)
	//tree.Put(19, 19)
	//tree.Put(20, 20)
	//tree.Put(21, 21)
	//tree.Put(22, 22)
	//tree.Put(23, 23)
	//tree.Put(24, 24)
	//tree.Put(25, 25)
	//tree.Put(26, 26)
	//tree.Put(27, 27)
	//
	//t.Log(tree.String())
	//tree.Remove(20)
	//t.Log(tree.String())
	//tree.Remove(11)
	//t.Log(tree.String())
	//tree.Remove(12)
	//t.Log(tree.String())
	//tree.Remove(26)
	//t.Log(tree.String())
	//tree.Remove(1)
	//t.Log(tree.String())
	//tree.Remove(0)
	//t.Log(tree.String())
	//tree.Remove(27)
	//t.Log(tree.String())
	//tree.Remove(9)
	//t.Log(tree.String())

	var wg = sync.WaitGroup{}
	wg.Add(400)
	for i := 0; i < 200; i++ {
		ni := i
		go func() {
			tree.Put(ni, ni)
			wg.Done()
		}()
		go func() {
			if ni%2 == 0 {
				tree.Remove(ni)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	t.Log(tree.String())

	//wg.Add(200)
	//for i := 0; i < 200; i++ {
	//	ni := i
	//	go func() {
	//		if rand.Intn(10) > 4 {
	//			tree.Remove(ni)
	//		}
	//		wg.Done()
	//	}()
	//}
	//wg.Wait()

	//t.Log(tree.String())
}

func TestDebug(t *testing.T) {
	tree := NewWith[int](3, utils.IntComparator[int])
	tree.Put(10, 10)
	tree.Put(2, 2)
	tree.Put(19, 19)
	tree.Put(0, 0)
	tree.Put(3, 3)
	tree.Put(15, 15)
	tree.Put(1, 1)
	tree.Put(4, 4)
	tree.Put(5, 5)
	tree.Put(17, 17)
	tree.Put(6, 6)
	tree.Put(14, 14)
	tree.Put(7, 7)
	tree.Put(12, 12)
	tree.Put(16, 16)
	tree.Put(9, 9)
	tree.Put(8, 8)
	tree.Put(13, 13)
	tree.Put(18, 18)
	tree.Put(11, 11)

	fmt.Println(tree.String())
	tree.Remove(0)
	fmt.Println(tree.String())
	tree.Remove(19)
	fmt.Println(tree.String())
	tree.Remove(1)
	fmt.Println(tree.String())
	tree.Remove(4)
	fmt.Println(tree.String())
	tree.Remove(17)
	fmt.Println(tree.String())
	tree.Remove(15)
	fmt.Println(tree.String())
	tree.Remove(18)
	fmt.Println(tree.String())
	tree.Remove(8)
	fmt.Println(tree.String())
	tree.Remove(10)
	fmt.Println(tree.String())
	tree.Remove(2)
	fmt.Println(tree.String())
	tree.Remove(14)
	fmt.Println(tree.String())
}

func BenchmarkPut(b *testing.B) {
	//mux := sync.Mutex{}
	tree := NewWith[int](3, utils.IntComparator[int])
	//tree.Put(1, 1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			//mux.Lock()
			tree.Put(1, 1)
			//tree.Remove(1)
			//mux.Unlock()
		}
	})
}

func TestRange(t *testing.T) {
	tree := NewWith[int](3, utils.IntComparator[int])
	tree.Put(10, 10)
	tree.Put(2, 2)
	tree.Put(19, 19)
	tree.Put(0, 0)
	tree.Put(3, 3)
	tree.Put(15, 15)
	tree.Put(1, 1)
	tree.Put(4, 4)
	tree.Put(5, 5)
	tree.Put(17, 17)
	tree.Put(6, 6)
	tree.Put(14, 14)
	tree.Put(7, 7)
	tree.Put(12, 12)
	tree.Put(16, 16)
	tree.Put(9, 9)
	tree.Put(8, 8)
	tree.Put(13, 13)
	tree.Put(18, 18)
	tree.Put(11, 11)

	fmt.Println(tree.String())

	values := tree.Range(2, 10)
	fmt.Println(values)
}
