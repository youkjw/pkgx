package timingwheel

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestSetTask(t *testing.T) {
	tm := NewTimingWheel[int](time.Second, 60, func(key int, value any) {
		t.Log(fmt.Sprintf("key:%d value:%v", key, value))
	})
	for i := 0; i < 10000; i++ {
		tm.SetTimer(i, i, time.Duration(rand.Intn(8))*time.Second)
	}
	time.Sleep(10 * time.Second)
}
