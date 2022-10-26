package timingwheel

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

const worker = 8

var ErrNotRunning = errors.New("TimingWheel is NotRunning")

type Execute[K any, V any] func(key K, value V)

// TimingWheel 时间轮
type TimingWheel[K any, V any] struct {
	interval       time.Duration // 时间轮区间时间长度
	timer          *time.Ticker  // 时间轮区间执行间隔定时器
	tickedPos      int           // 当前执行的时间轮区
	slots          []*list.List  // 时间轮单budget区间列表
	numSlots       int           // 时间轮budget数
	execute        Execute[K, V]
	setChannel     chan *timingEntry[K, V]
	removeChannel  chan K
	runningChannel chan *timingEntry[K, V]
	stopChannel    chan bool
	taskPosition   sync.Map
	running        bool
}

type timingEntry[K any, V any] struct {
	baseEntry[K]
	value   V
	circle  int // 圈数
	removed bool
}

type baseEntry[K any] struct {
	baseTime time.Duration
	delay    time.Duration // 延迟时间
	key      K
}

type positionEntry[K any, V any] struct {
	pos  int
	item *timingEntry[K, V]
}

func NewTimingWheel[K any, V any](interval time.Duration, numSlots int, execute Execute[K, V]) *TimingWheel[K, V] {
	tw := &TimingWheel[K, V]{
		interval:       interval,
		timer:          time.NewTicker(interval),
		numSlots:       numSlots,
		tickedPos:      0, // 从区间0开始计算时间轮
		slots:          make([]*list.List, numSlots),
		execute:        execute,
		setChannel:     make(chan *timingEntry[K, V]),
		removeChannel:  make(chan K),
		runningChannel: make(chan *timingEntry[K, V], worker),
		stopChannel:    make(chan bool),
	}

	tw.initSlots()
	go tw.run()
	tw.running = true
	return tw
}

func (tw *TimingWheel[K, V]) initSlots() {
	for i := 0; i < tw.numSlots; i++ {
		tw.slots[i] = list.New()
	}
}

func (tw *TimingWheel[K, V]) SetTimer(key K, value V, delay time.Duration) error {
	if !tw.running {
		return ErrNotRunning
	}

	task := &timingEntry[K, V]{
		baseEntry: baseEntry[K]{
			baseTime: time.Duration(time.Now().Nanosecond()),
			delay:    delay,
			key:      key,
		},
		value: value,
	}

	if delay < tw.interval {
		tw.push(task)
		return nil
	}

	tw.setChannel <- task
	return nil
}

func (tw *TimingWheel[K, V]) RemoveTimer(key K) error {
	if !tw.running {
		return ErrNotRunning
	}

	tw.removeChannel <- key
	return nil
}

func (tw *TimingWheel[K, V]) run() {
	tw.workerStart()
	for {
		select {
		case <-tw.timer.C:
			tw.onTick()
		case task := <-tw.setChannel:
			tw.setTask(task)
		case key := <-tw.removeChannel:
			tw.removeTask(key)
		case <-tw.stopChannel:
			tw.timer.Stop()
		}
	}
}

func (tw *TimingWheel[K, V]) onTick() {
	tw.tickedPos = (tw.tickedPos + 1) % tw.numSlots
	l := tw.slots[tw.tickedPos]
	tw.scanAndRunTasks(l)
}

func (tw *TimingWheel[K, V]) scanAndRunTasks(list *list.List) {
	for e := list.Front(); e != nil; {
		task := e.Value.(*timingEntry[K, V])
		if task.removed {
			next := e.Next()
			list.Remove(e)
			e = next
			continue
		} else if time.Now().After(time.Unix(0, task.baseTime.Nanoseconds()+task.delay.Nanoseconds())) {
			goto RUN
		} else if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		}
	RUN:
		tw.push(task)
		next := e.Next()
		list.Remove(e)
		tw.taskPosition.Delete(task.key)
		e = next
	}
}

func (tw *TimingWheel[K, V]) setTask(task *timingEntry[K, V]) {
	if val, ok := tw.taskPosition.Load(task.key); ok {
		timer := val.(*positionEntry[K, V])
		timer.item.value = task.value
		tw.moveTask(task.baseEntry)
	} else {
		pos, circle := tw.getPositionAndCircle(task.delay)
		task.circle = circle
		tw.slots[pos].PushBack(task)
		tw.setTimerPosition(pos, task)
	}
}

func (tw *TimingWheel[K, V]) moveTask(base baseEntry[K]) {
	val, ok := tw.taskPosition.Load(base.key)
	if !ok {
		return
	}

	timer := val.(*positionEntry[K, V])
	if base.delay < tw.interval {
		tw.push(&timingEntry[K, V]{
			baseEntry: base,
			value:     timer.item.value,
			circle:    0,
		})
		return
	}

	pos, circle := tw.getPositionAndCircle(base.delay)
	if pos != timer.pos {
		timer.item.removed = true
		newTimer := &timingEntry[K, V]{
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

func (tw *TimingWheel[K, V]) push(task *timingEntry[K, V]) {
	tw.runningChannel <- task
}

func (tw *TimingWheel[K, V]) workerStart() {
	for i := 0; i < worker; i++ {
		go func() {
			for {
				select {
				case task := <-tw.runningChannel:
					go func() {
						tw.execute(task.key, task.value)
					}()
				}

			}
		}()
	}
}

func (tw *TimingWheel[K, V]) removeTask(key K) {
	val, ok := tw.taskPosition.Load(key)
	if !ok {
		return
	}

	timer := val.(*positionEntry[K, V])
	timer.item.removed = true
	tw.taskPosition.Delete(key)
}

func (tw *TimingWheel[K, V]) setTimerPosition(pos int, task *timingEntry[K, V]) {
	if val, ok := tw.taskPosition.Load(task.key); ok {
		timer := val.(*positionEntry[K, V])
		timer.item = task
		timer.pos = pos
	} else {
		tw.taskPosition.Store(task.key, &positionEntry[K, V]{
			pos:  pos,
			item: task,
		})
	}
}

func (tw *TimingWheel[K, V]) getPositionAndCircle(d time.Duration) (pos, circle int) {
	steps := int(d / tw.interval)
	pos = (tw.tickedPos + steps) % tw.numSlots
	circle = (tw.tickedPos + steps) / tw.numSlots
	return
}

func (tw *TimingWheel[K, V]) Close() {
	tw.stopChannel <- true
	tw.running = false
}
