package timingwheel

import (
	"fmt"
	"testing"
	"time"
)

func TestSetTask(t *testing.T) {
	tm := NewTimingWheel[int](time.Second, 60, func(key int, value any) {
		t.Log(fmt.Sprintf("key:%d value:%v", key, value))
	})
	err := tm.SetTimer(1, 111, 5*time.Second)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(10 * time.Second)
}
