package ik

import (
	"sync"
	"reflect"
	"fmt"
)

type descriptorListHead struct {
	next *spawneeDescriptor
	prev *spawneeDescriptor
}

type descriptorList struct {
	count int
	first *spawneeDescriptor
	last  *spawneeDescriptor
}

type spawneeDescriptor struct {
	head_alive        descriptorListHead
	head_dead         descriptorListHead
	id                int
	spawnee           Spawnee
	exitStatus        error
	shutdownRequested bool
	mtx               sync.Mutex
	cond              *sync.Cond
}

type SpawneeStatus struct {
	Id                int
	Spawnee           Spawnee
	ExitStatus        error
}

type dispatchReturnValue struct {
	b bool
	s []Spawnee
	e error
	ss []SpawneeStatus
}

type dispatch struct {
	fn      func(Spawnee, chan dispatchReturnValue)
	spawnee Spawnee
	retval  chan dispatchReturnValue
}

const (
	Spawned = 1
	Stopped = 2
)

type ContinueType struct{}

type Panicked struct {
	panic interface{}
}

func (_ *ContinueType) Error() string { return "" }

func typeName(type_ reflect.Type) string {
	if type_.Kind() == reflect.Ptr {
		return "*" + typeName(type_.Elem())
	} else {
		return type_.Name()
	}
}

func (panicked *Panicked) Error() string {
	switch panic_ := panicked.panic.(type) {
	case string:
		return panic_
	case error:
		return fmt.Sprintf("(%s) %s", typeName(reflect.TypeOf(panic_)), panic_.Error())
	default:
		type_ := reflect.TypeOf(panic_)
		method, ok := type_.MethodByName("String")
		if ok && method.Type.NumIn() == 1 {
			result := method.Func.Call([]reflect.Value { reflect.ValueOf(panic_) })
			if len(result) == 1 && result[0].Type().Kind() == reflect.String {
				return fmt.Sprintf("(%s) %s", typeName(type_), result[0].String())
			}
		}
		return fmt.Sprintf("(%s)", typeName(type_))
	}
}

var Continue = &ContinueType{}

type NotFoundType struct{}

func (_ *NotFoundType) Error() string { return "not found" }

var NotFound = &NotFoundType{}

type spawnerEvent struct {
	t       int
	spawnee Spawnee
}

type Spawner struct {
	alives    descriptorList
	deads     descriptorList
	m         map[Spawnee]*spawneeDescriptor
	c         chan dispatch
	mtx       sync.Mutex
	cond      *sync.Cond
	lastEvent *spawnerEvent
}

func newDescriptor(spawnee Spawnee, id int) *spawneeDescriptor {
	retval := &spawneeDescriptor{
		head_alive:        descriptorListHead{nil, nil},
		head_dead:         descriptorListHead{nil, nil},
		id:                id,
		spawnee:           spawnee,
		exitStatus:        Continue,
		shutdownRequested: false,
		mtx:               sync.Mutex{},
		cond:              nil,
	}
	retval.cond = sync.NewCond(&retval.mtx)
	return retval
}

func (spawner *Spawner) spawn(spawnee Spawnee, retval chan dispatchReturnValue) {
	go func() {
		descriptor := newDescriptor(spawnee, len(spawner.m) + 1)
		func() {
			spawner.mtx.Lock()
			defer spawner.mtx.Unlock()
			if spawner.alives.last != nil {
				spawner.alives.last.head_alive.next = descriptor
				descriptor.head_alive.prev = spawner.alives.last
			}
			if spawner.alives.first == nil {
				spawner.alives.first = descriptor
			}
			spawner.alives.last = descriptor
			spawner.alives.count += 1
			spawner.m[spawnee] = descriptor
			// notify the event
			spawner.lastEvent = &spawnerEvent{
				t:       Spawned,
				spawnee: spawnee,
			}
			spawner.cond.Broadcast()
			retval <- dispatchReturnValue{true, nil, nil, nil}
		}()
		var exitStatus error = nil
		func() {
			defer func() {
				r := recover()
				if r != nil {
					exitStatus = &Panicked { r }
				}
			}()
			exitStatus = Continue
			for exitStatus == Continue {
				exitStatus = descriptor.spawnee.Run()
			}
		}()
		func() {
			spawner.mtx.Lock()
			defer spawner.mtx.Unlock()
			descriptor.exitStatus = exitStatus
			// remove from alive list
			if descriptor.head_alive.prev != nil {
				descriptor.head_alive.prev.head_alive.next = descriptor.head_alive.next
			} else {
				spawner.alives.first = descriptor.head_alive.next
			}
			if descriptor.head_alive.next != nil {
				descriptor.head_alive.next.head_alive.prev = descriptor.head_alive.prev
			} else {
				spawner.alives.last = descriptor.head_alive.prev
			}
			spawner.alives.count -= 1
			// append to dead list
			if spawner.deads.last != nil {
				spawner.deads.last.head_dead.next = descriptor
				descriptor.head_dead.prev = spawner.deads.last
			}
			if spawner.deads.first == nil {
				spawner.deads.first = descriptor
			}
			spawner.deads.last = descriptor
			spawner.deads.count += 1

			// notify the event
			spawner.lastEvent = &spawnerEvent{
				t:       Stopped,
				spawnee: spawnee,
			}
			spawner.cond.Broadcast()
			descriptor.cond.Broadcast()
		}()
	}()
}

func (spawner *Spawner) kill(spawnee Spawnee, retval chan dispatchReturnValue) {
	spawner.mtx.Lock()
	descriptor, ok := spawner.m[spawnee]
	spawner.mtx.Unlock()
	if ok && descriptor.exitStatus != Continue {
		descriptor.shutdownRequested = true
		err := spawnee.Shutdown()
		retval <- dispatchReturnValue{true, nil, err, nil}
	} else {
		retval <- dispatchReturnValue{false, nil, nil, nil}
	}
}

func (spawner *Spawner) getStatus(spawnee Spawnee, retval chan dispatchReturnValue) {
	spawner.mtx.Lock()
	defer spawner.mtx.Unlock()
	descriptor, ok := spawner.m[spawnee]
	if ok {
		retval <- dispatchReturnValue{false, nil, descriptor.exitStatus, nil}
	} else {
		retval <- dispatchReturnValue{false, nil, nil, nil}
	}
}

func (spawner *Spawner) getRunningSpawnees(_ Spawnee, retval chan dispatchReturnValue) {
	spawner.mtx.Lock()
	defer spawner.mtx.Unlock()
	spawnees := make([]Spawnee, spawner.alives.count)
	descriptor := spawner.alives.first
	i := 0
	for descriptor != nil {
		spawnees[i] = descriptor.spawnee
		descriptor = descriptor.head_alive.next
		i += 1
	}
	retval <- dispatchReturnValue{false, spawnees, nil, nil}
}

func (spawner *Spawner) getStoppedSpawnees(_ Spawnee, retval chan dispatchReturnValue) {
	spawner.mtx.Lock()
	defer spawner.mtx.Unlock()
	spawnees := make([]Spawnee, spawner.deads.count)
	descriptor := spawner.deads.first
	i := 0
	for descriptor != nil {
		spawnees[i] = descriptor.spawnee
		descriptor = descriptor.head_dead.next
		i += 1
	}
	retval <- dispatchReturnValue{false, spawnees, nil, nil}
}

func (spawner *Spawner) getSpawneeStatuses(_ Spawnee, retval chan dispatchReturnValue) {
	spawner.mtx.Lock()
	defer spawner.mtx.Unlock()
	spawneeStatuses := make([]SpawneeStatus, len(spawner.m))
	i := 0
	for spawnee, descriptor := range spawner.m {
		spawneeStatuses[i] = SpawneeStatus {Id: descriptor.id, Spawnee: spawnee, ExitStatus: descriptor.exitStatus}
		i += 1
	}
	retval <- dispatchReturnValue{false, nil, nil, spawneeStatuses}
}


func (spawner *Spawner) Spawn(spawnee Spawnee) error {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.spawn, spawnee, retval}
	retval_ := <-retval
	return retval_.e
}

func (spawner *Spawner) Kill(spawnee Spawnee) (bool, error) {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.kill, spawnee, retval}
	retval_ := <-retval
	return retval_.b, retval_.e
}

func (spawner *Spawner) GetStatus(spawnee Spawnee) error {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.getStatus, spawnee, retval}
	retval_ := <-retval
	return retval_.e
}

func (spawner *Spawner) GetRunningSpawnees() ([]Spawnee, error) {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.getRunningSpawnees, nil, retval}
	retval_ := <-retval
	return retval_.s, retval_.e
}

func (spawner *Spawner) GetStoppedSpawnees() ([]Spawnee, error) {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.getStoppedSpawnees, nil, retval}
	retval_ := <-retval
	return retval_.s, retval_.e
}

func (spawner *Spawner) GetSpawneeStatuses() ([]SpawneeStatus, error) {
	retval := make(chan dispatchReturnValue)
	spawner.c <- dispatch{spawner.getSpawneeStatuses, nil, retval}
	retval_ := <-retval
	return retval_.ss, retval_.e
}

func (spawner *Spawner) Poll(spawnee Spawnee) error {
	descriptor, ok := spawner.m[spawnee]
	if !ok {
		return NotFound
	}
	if func() bool {
		spawner.mtx.Lock()
		defer spawner.mtx.Unlock()
		if descriptor.exitStatus != Continue {
			return true
		}
		return false
	}() {
		return nil
	}
	defer descriptor.mtx.Unlock()
	descriptor.mtx.Lock()
	descriptor.cond.Wait()
	return nil
}

func (spawner *Spawner) PollMultiple(spawnees []Spawnee) error {
	spawnees_ := make(map[Spawnee]bool)
	for _, spawnee := range spawnees {
		spawnees_[spawnee] = true
	}
	count := len(spawnees_)
	for count > 0 {
		spawner.mtx.Lock()
		spawner.cond.Wait()
		lastEvent := spawner.lastEvent
		spawner.mtx.Unlock()
		if lastEvent.t == Stopped {
			spawnee := lastEvent.spawnee
			if alive, ok := spawnees_[spawnee]; alive && ok {
				spawnees_[spawnee] = false
				count -= 1
			}
		}
	}
	return nil
}

func NewSpawner() *Spawner {
	c := make(chan dispatch)
	// launch the supervisor
	go func() {
		for {
			select {
			case disp := <-c:
				disp.fn(disp.spawnee, disp.retval)
			}
		}
	}()
	spawner := &Spawner{
		alives:    descriptorList{0, nil, nil},
		deads:     descriptorList{0, nil, nil},
		m:         make(map[Spawnee]*spawneeDescriptor),
		c:         c,
		mtx:       sync.Mutex{},
		cond:      nil,
		lastEvent: nil,
	}
	spawner.cond = sync.NewCond(&spawner.mtx)
	return spawner
}
