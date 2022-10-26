package timingwheel

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestSetTask(t *testing.T) {
	tm := NewTimingWheel[int, string](time.Second, 60, func(key int, value string) {
		t.Log(fmt.Sprintf("key:%d value:%v", key, value))
	})
	for i := 0; i < 100; i++ {
		tm.SetTimer(i, "a", time.Duration(rand.Intn(8))*time.Second)
	}
	time.Sleep(10 * time.Second)
}
