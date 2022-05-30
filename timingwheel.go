package memory

import (
	"container/list"
	"context"
	"time"
)

type (
	// Execute defines the method to execute the task.
	Execute func(key, value interface{})

	timingwheel struct {
		ctx context.Context

		interval time.Duration
		execute  Execute
		numSlots int
		slots    []*list.List
	}
)

func instanceTimingWheel(ctx context.Context) *timingwheel {
	t := &timingwheel{
		ctx:      nil,
		interval: 0,
		execute:  nil,
		numSlots: 0,
		slots:    nil,
	}
	t.initSlots()
	return t
}

func (tw *timingwheel) initSlots() {
	for i := 0; i < tw.numSlots; i++ {
		tw.slots[i] = list.New()
	}
}
