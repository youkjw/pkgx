package list

import "testing"

func BenchmarkArrayListAdd(b *testing.B) {
	var s = New[int]()
	for i := 0; i < 1000000; i++ {
		var tmp = i
		s.Add(tmp)
	}
}

func BenchmarkSliceAdd(b *testing.B) {
	var s = make([]int, 0)
	for i := 0; i < 1000000; i++ {
		var tmp = i
		s = append(s, tmp)
	}
}
