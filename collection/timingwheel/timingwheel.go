package timingwheel

import (
	"container/list"
	"sync"
	"time"
)

const worker = 8

type Execute[V any] func(key V, value any)

// TimingWheel 时间轮
type TimingWheel[V any] struct {
	interval       time.Duration // 时间轮区间时间长度
	timer          *time.Ticker  // 时间轮区间执行间隔定时器
	tickedPos      int           // 当前执行的时间轮区
	slots          []*list.List  // 时间轮单budget区间列表
	numSlots       int           // 时间轮budget数
	execute        Execute[V]
	setChannel     chan *timingEntry[V]
	runningChannel chan *timingEntry[V]
	taskPosition   sync.Map
}

type timingEntry[V any] struct {
	baseEntry[V]
	value   any
	circle  int // 圈数
	removed bool
}

type baseEntry[V any] struct {
	nt    time.Duration
	delay time.Duration // 延迟时间
	key   V
}

type positionEntry[V any] struct {
	pos  int
	item *timingEntry[V]
}

func NewTimingWheel[V any](interval time.Duration, numSlots int, execute Execute[V]) {
	tw := &TimingWheel[V]{
		interval:       interval,
		timer:          time.NewTicker(interval),
		numSlots:       numSlots,
		tickedPos:      0, // 从区间0开始计算时间轮
		slots:          make([]*list.List, numSlots),
		execute:        execute,
		setChannel:     make(chan *timingEntry[V]),
		runningChannel: make(chan *timingEntry[V]),
	}

	tw.initSlots()
	go tw.run()
}

func (tw *TimingWheel[V]) initSlots() {
	for i := 0; i < tw.numSlots; i++ {
		tw.slots[i] = list.New()
	}
}

func (tw *TimingWheel[V]) setTimer(key V, value any, delay time.Duration) error {
	if delay < tw.interval {
		delay = tw.interval
	}

	tw.setChannel <- &timingEntry[V]{
		baseEntry: baseEntry[V]{
			nt:    time.Duration(time.Now().Unix()),
			delay: delay,
			key:   key,
		},
		value: value,
	}
	return nil
}

func (tw *TimingWheel[V]) run() {
	for {
		select {
		case <-tw.timer.C:
			tw.onTick()
		case task := <-tw.setChannel:
			tw.setTask(task)
		}
	}
}

func (tw *TimingWheel[V]) onTick() {
	tw.tickedPos = tw.tickedPos % tw.numSlots
}

func (tw *TimingWheel[V]) setTask(task *timingEntry[V]) {
	if val, ok := tw.taskPosition.Load(task.key); ok {
		timer := val.(*positionEntry[V])
		timer.item.value = task.value
		tw.moveTask(task.baseEntry)
	} else {
		pos, circle := tw.getPositionAndCircle(task.delay)
		task.circle = circle
		tw.slots[pos].PushBack(task)
		tw.setTimerPosition(pos, task)
	}
}

func (tw *TimingWheel[V]) moveTask(base baseEntry[V]) {
	val, ok := tw.taskPosition.Load(base.key)
	if !ok {
		return
	}

	timer := val.(*positionEntry[V])
	if base.delay < tw.interval {
		tw.exec(timer.item.key, timer.item.value)
	}

	pos, circle := tw.getPositionAndCircle(base.delay)
	if pos != timer.pos {
		timer.item.removed = true
		newTimer := &timingEntry[V]{
			baseEntry: base,
			value:     timer.item.value,
			circle:    circle,
		}
		tw.slots[pos].PushBack(newTimer)
		tw.setTimerPosition(pos, newTimer)
	} else if circle > 0 {
		timer.item.circle = circle
	}
}

func (tw *TimingWheel[V]) exec(key V, value any) {
	go func() {
		tw.execute(key, value)
	}()
	return
}

func (tw *TimingWheel[V]) setTimerPosition(pos int, task *timingEntry[V]) {
	if val, ok := tw.taskPosition.Load(task.key); ok {
		timer := val.(*positionEntry[V])
		timer.item = task
		timer.pos = pos
	} else {
		tw.taskPosition.Store(task.key, &positionEntry[V]{
			pos:  pos,
			item: task,
		})
	}
}

func (tw *TimingWheel[V]) getPositionAndCircle(d time.Duration) (pos, circle int) {
	steps := int(d / tw.interval)
	pos = (tw.tickedPos + steps) % tw.numSlots
	circle = (tw.tickedPos + steps) / tw.numSlots
	return
}
