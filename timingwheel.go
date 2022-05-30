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
		ticker   *time.Ticker
		execute  Execute
		numSlots int
		slots    []*list.List
	}
)

func instanceTimingWheel(ctx context.Context, interval time.Duration) *timingwheel {
	tw := &timingwheel{
		ctx:      nil,
		interval: interval,
		ticker:   time.NewTicker(interval),
		execute:  nil,
		numSlots: 300,
		slots:    nil,
	}
	tw.initSlots()
	go tw.run()
	return tw
}

func (tw *timingwheel) initSlots() {
	for i := 0; i < tw.numSlots; i++ {
		tw.slots[i] = list.New()
	}
}

func (tw *timingwheel) run() {
TASK:
	for {
		select {
		case <-tw.ticker.C:

		case <-tw.ctx.Done():
			break TASK
		}
	}
}
