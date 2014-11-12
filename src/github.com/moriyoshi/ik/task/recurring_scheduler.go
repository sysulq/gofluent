package task

import (
	"time"
	"sort"
	"errors"
	"sync/atomic"
)

type RecurringTaskSpec struct {
	month []int
	dayOfWeek []int
	dayOfMonth []int
	hour []int
	minute []int
	rightAt time.Time
}

type RecurringTaskDescriptor struct {
	id int64
	spec RecurringTaskSpec
	nextTime time.Time
	status int
	fn func (int64, time.Time, *RecurringTaskSpec) (interface {}, error)
}

type RecurringTaskDescriptorHeap []*RecurringTaskDescriptor

type RecurringTaskTimeResolution int

const (
	Insert = iota
	Update
	TryPop
	Delete
	NoOp
)

const (
	Stopped = 0
	Running = 1
)

const (
	Nanosecond = RecurringTaskTimeResolution(0)
	Second     = RecurringTaskTimeResolution(1)
	Minute     = RecurringTaskTimeResolution(2)
	Hour       = RecurringTaskTimeResolution(3)
	Day        = RecurringTaskTimeResolution(4)
	Month      = RecurringTaskTimeResolution(5)
	Year       = RecurringTaskTimeResolution(6)
)

type RecurringTaskDaemonCommandResult struct {
	descriptor *RecurringTaskDescriptor
	diff time.Duration
}

type RecurringTaskDaemonCommand struct {
	command int
	descriptor *RecurringTaskDescriptor
	time time.Time
	result chan RecurringTaskDaemonCommandResult
}

type RecurringTaskScheduler struct {
	pQueue RecurringTaskDescriptorHeap
	nowGetter func () time.Time
	taskRunner TaskRunner
	daemonChan chan RecurringTaskDaemonCommand
	nextId int64
}

func (heap *RecurringTaskDescriptorHeap) insert(elem *RecurringTaskDescriptor) {
	l := len(*heap)
	if cap(*heap) < l + 1 {
		newCap := cap(*heap) * 2
		if newCap < cap(*heap) { panic("?") }
		newHeap := make([]*RecurringTaskDescriptor, newCap, l + 1)
		copy(newHeap, *heap)
		*heap = newHeap
	} else {
		*heap = (*heap)[0:l + 1]
	}
	(*heap)[l] = elem
	heap.bubbleUp(l)
}

func (heap *RecurringTaskDescriptorHeap) find(elem *RecurringTaskDescriptor) int {
	_heap := *heap
	for i := 0; i < len(_heap); i += 1 {
		if _heap[i] == elem {
			return i
		}
	}
	return -1
}

func (heap *RecurringTaskDescriptorHeap) update(elem *RecurringTaskDescriptor) {
	i := heap.find(elem)
	if i < 0 {
		panic("should never happen")
	}
	_heap := *heap
	copy(_heap[i:], _heap[i + 1:])
	i = len(_heap) - 1
	_heap[i] = elem
	heap.bubbleUp(i)
}

func (heap *RecurringTaskDescriptorHeap) delete(elem *RecurringTaskDescriptor) {
	i := heap.find(elem)
	if i < 0 {
		panic("should never happen")
	}
	_heap := *heap
	copy(_heap[i:], _heap[i + 1:])
	// readjust the capacity
	*heap = _heap[0:len(_heap) - 1]
}

func (heap *RecurringTaskDescriptorHeap) bubbleUp(i int) {
	for i > 0 {
		node := &(*heap)[i]
		parentIndex := (i - 1) >> 1
		parentNode := &(*heap)[parentIndex]
		if (*parentNode).status != Running && (*node).status == Running {
			break
		} else if ((*parentNode).status == Running && (*node).status != Running) || (*parentNode).nextTime.After((*node).nextTime) {
			*node, *parentNode = *parentNode, *node
			i = parentIndex
		} else {
			break
		}
	}
}

func (spec *RecurringTaskSpec) copy() RecurringTaskSpec {
	retval := RecurringTaskSpec {}
	if spec.month != nil {
		retval.month = make([]int, len(spec.month))
		copy(retval.month, spec.month)
	}
	if spec.dayOfMonth != nil {
		retval.dayOfMonth = make([]int, len(spec.dayOfMonth))
		copy(retval.dayOfMonth, spec.dayOfMonth)
	}
	if spec.dayOfWeek != nil {
		retval.dayOfWeek = make([]int, len(spec.dayOfWeek))
		copy(retval.dayOfWeek, spec.dayOfWeek)
	}
	if spec.hour != nil {
		retval.hour = make([]int, len(spec.hour))
		copy(retval.hour, spec.hour)
	}
	if spec.minute != nil {
		retval.minute = make([]int, len(spec.minute))
		copy(retval.minute, spec.minute)
	}
	retval.rightAt = spec.rightAt
	return retval
}

func (spec *RecurringTaskSpec) ensureSorted() {
	sort.IntSlice(spec.month).Sort()
	sort.IntSlice(spec.dayOfMonth).Sort()
	sort.IntSlice(spec.dayOfWeek).Sort()
	sort.IntSlice(spec.hour).Sort()
	sort.IntSlice(spec.minute).Sort()
}

func (spec *RecurringTaskSpec) isZero() bool {
	return spec.month == nil && spec.dayOfMonth == nil && spec.dayOfWeek == nil && spec.hour == nil && spec.minute == nil && spec.rightAt.IsZero()
}

type timeStruct struct {
	second int
	minute int
	hour int
	day int
	month time.Month
	year int
	location *time.Location
}

func newTimeStruct(t time.Time) timeStruct {
	currentYear, currentMonth, currentDay := t.Date()
	currentHour, currentMinute, currentSecond := t.Clock()
	return timeStruct {
		second: currentSecond,
		minute: currentMinute,
		hour: currentHour,
		day: currentDay,
		month: currentMonth,
		year: currentYear,
		location: t.Location(),
	}
}

func (tm *timeStruct) toTime() time.Time {
	return time.Date(tm.year, tm.month, tm.day, tm.hour, tm.minute, tm.second, 0, tm.location)
}

func daysIn(month time.Month, year int) int {
	return time.Date(year, month + 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Hour).Day()
}

func (spec *RecurringTaskSpec) resolution() RecurringTaskTimeResolution {
	if !spec.rightAt.IsZero() {
		return Nanosecond
	}
	if spec.minute != nil {
		return Minute
	}
	if spec.hour != nil {
		return Hour
	}
	if spec.dayOfWeek != nil || spec.dayOfMonth != nil {
		return Day
	}
	if spec.month != nil {
		return Month
	}
	{
		panic("should never get here")
	}
}

func incrementByResolution(t time.Time, res RecurringTaskTimeResolution, amount int) time.Time {
	switch res {
	case Nanosecond:
		return t.Add(time.Duration(amount))
	case Second:
		return t.Add(time.Duration(amount * 1000000000))
	case Minute:
		return t.Add(time.Duration(amount * 1000000000 * 60))
	case Hour:
		return t.Add(time.Duration(amount * 1000000000 * 60 * 60))
	case Day:
		return t.AddDate(0, 0, amount)
	case Month:
		return t.AddDate(0, amount, 0)
	case Year:
		return t.AddDate(amount, 0, 0)
	default:
		panic("should never get here")
	}
}

func (spec *RecurringTaskSpec) nextTime(now time.Time) (time.Time, error) {
	tm := newTimeStruct(now)
	next := timeStruct { 0, -1, -1, -1, -1, tm.year, tm.location }

	if !spec.rightAt.IsZero() {
		return spec.rightAt, nil
	}

	if spec.minute != nil {
		if len(spec.minute) == 0 {
			return time.Time {}, errors.New("invalid time spec")
		}
		for _, minute := range spec.minute {
			if minute >= tm.minute {
				next.minute = minute
				break
			}
		}
		if next.minute < 0 {
			next.minute = spec.minute[0]
			tm.hour += 1 // carry
			if tm.hour >= 24 {
				tm.hour -= 24
				tm.day += 1
			}
		}
	} else {
		next.minute = tm.minute
	}

	if spec.hour != nil {
		if len(spec.hour) == 0 {
			return time.Time {}, errors.New("invalid time spec")
		}
		for _, hour := range spec.hour {
			if hour >= tm.hour {
				next.hour = hour
				break
			}
		}
		if next.hour < 0 {
			next.hour = spec.hour[0]
			tm.day += 1 // carry
		}
	} else {
		next.hour = tm.hour
	}

	daysInMonth := daysIn(tm.month, tm.year)
	if tm.day > daysInMonth {
		tm.day -= daysInMonth
		tm.month += 1
		if tm.month > 12 {
			tm.month -= 12
			tm.year += 1
		}
		daysInMonth = daysIn(tm.month, tm.year)
	}

	// if both dayOfWeek and dayOfMonth are specified, the next execution
	// time will be either one that comes first (according to crontab(5))

	nextDayCandidate1 := -1
	if spec.dayOfWeek != nil {
		if len(spec.dayOfWeek) == 0 {
			return time.Time {}, errors.New("invalid time spec")
		}
		currentDayOfWeek := (&tm).toTime().Weekday()
		dayOfWeek := -1
		for _, dayOfWeek_ := range spec.dayOfWeek {
			if dayOfWeek_ >= int(currentDayOfWeek) {
				dayOfWeek = dayOfWeek_
			}
		}
		if dayOfWeek < 0 {
			nextDayCandidate1 = tm.day + spec.dayOfWeek[0] + 7 - int(currentDayOfWeek)
		} else {
			nextDayCandidate1 = tm.day + dayOfWeek - int(currentDayOfWeek)
		}
	}

	nextDayCandidate2 := -1
	if spec.dayOfMonth != nil {
		if len(spec.dayOfMonth) == 0 {
			return time.Time {}, errors.New("invalid time spec")
		}
		for _, dayOfMonth := range spec.dayOfMonth {
			if dayOfMonth >= tm.day {
				nextDayCandidate2 = dayOfMonth
				break
			}
		}
		if nextDayCandidate2 < 0 {
			nextDayCandidate2 = daysInMonth + spec.dayOfMonth[0] - 1
		}
	}

	if nextDayCandidate1 == -1 && nextDayCandidate2 == -1 {
		// do nothing
	} else if nextDayCandidate1 == -1 || nextDayCandidate1 > nextDayCandidate2 {
		tm.day = nextDayCandidate2
	} else {
		tm.day = nextDayCandidate1
	}

	if tm.day > daysInMonth {
		tm.day = tm.day - daysInMonth
		tm.month += 1
		if tm.month > 12 {
			tm.month -= 12
			tm.year += 1
		}
	}
	next.day = tm.day

	if spec.month != nil {
		if len(spec.dayOfMonth) == 0 {
			return time.Time {}, errors.New("invalid time spec")
		}
		for _, month := range spec.month {
			if month >= int(tm.month) {
				next.month = time.Month(month)
				break
			}
		}
		if next.month < 0 {
			next.month = time.Month(spec.month[0])
			tm.year += 1
		}
	} else {
		next.month = tm.month
	}

	return next.toTime(), nil
}

func (sched *RecurringTaskScheduler) RunNext() (time.Duration, TaskStatus, error) {
	now := sched.nowGetter()
	resultChan := make(chan RecurringTaskDaemonCommandResult)
	sched.daemonChan <- RecurringTaskDaemonCommand { TryPop, nil, now, resultChan }
	result := <-resultChan
	descr := result.descriptor
	remaining := result.diff
	if remaining > 0 {
		return remaining, nil, nil
	}
	taskStatus, err := sched.taskRunner.Run(func () (interface {}, error) {
		spec := (&descr.spec).copy()
		result, err := descr.fn(descr.id, now, &spec)
		if err != nil {
			return nil, err
		}
		if spec.isZero() {
			sched.daemonChan <- RecurringTaskDaemonCommand { Delete, descr, time.Time {}, nil }
		} else {
			_now := sched.nowGetter()
			res := (&spec).resolution()
			var nextTime time.Time
			for {
				nextTime, err = spec.nextTime(_now)
				if err != nil {
					return nil, err
				}
				if nextTime.Sub(descr.nextTime) > 0 {
					break
				}
				_now = incrementByResolution(_now, res, 1)
			}
			descr.spec = spec // XXX: hope this is safe
			sched.daemonChan <- RecurringTaskDaemonCommand { Update, descr, nextTime, nil }
		}
		return result, nil
	})
	return time.Duration(0), taskStatus, err
}

func (sched *RecurringTaskScheduler) NoOp() {
	sched.daemonChan <- RecurringTaskDaemonCommand { NoOp, nil, time.Time {}, nil }
}

func (sched *RecurringTaskScheduler) ProcessEvent() {
	cmd := <-sched.daemonChan
	switch cmd.command {
	case Insert:
		(&sched.pQueue).insert(cmd.descriptor)
	case Update:
		descr := cmd.descriptor
		descr.nextTime = cmd.time
		descr.status = Stopped
		sched.pQueue.update(descr)
		(&sched.pQueue).update(descr)
	case TryPop:
		descr := sched.pQueue[0]
		now := cmd.time
		diff := descr.nextTime.Sub(now)
		if diff <= 0 {
			descr.status = Running
			(&sched.pQueue).update(descr)
		}
		cmd.result <- RecurringTaskDaemonCommandResult { descr, diff }
	case Delete:
		(&sched.pQueue).delete(cmd.descriptor)
	case NoOp:
		// do nothing
	default:
		panic("WTF!")
	}
}

func (sched *RecurringTaskScheduler) _nextId() int64 {
	return atomic.AddInt64(&sched.nextId, 1)
}

func (sched *RecurringTaskScheduler) RegisterTask(spec RecurringTaskSpec, task func(int64, time.Time, *RecurringTaskSpec) (interface{}, error)) (int64, error) {
	spec = (&spec).copy()
	(&spec).ensureSorted()
	id := sched._nextId()
	descr := &RecurringTaskDescriptor {
		id: id,
		spec: spec,
		nextTime: time.Time {},
		status: Stopped,
		fn: task,
	}
	nextTime, err := spec.nextTime(sched.nowGetter())
	if err != nil {
		return -1, err
	}
	descr.nextTime = nextTime
	sched.daemonChan <- RecurringTaskDaemonCommand { Insert, descr, time.Time {}, nil }
	return id, nil
}

func NewRecurringTaskScheduler(nowGetter func() time.Time, taskRunner TaskRunner) *RecurringTaskScheduler {
	return &RecurringTaskScheduler {
		pQueue: make(RecurringTaskDescriptorHeap, 0, 16),
		nowGetter: nowGetter,
		taskRunner: taskRunner,
		daemonChan: make(chan RecurringTaskDaemonCommand, 1),
		nextId: 0,
	}
}
