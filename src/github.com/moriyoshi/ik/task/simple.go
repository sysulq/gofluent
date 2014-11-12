package task

import (
	"sync"
	"sync/atomic"
)

type SimpleTaskRunner struct {}

type simpleTaskStatus struct {
	condMutex sync.Mutex
	cond *sync.Cond
	completed uintptr
	status error
	result interface {}
}

func (status *simpleTaskStatus) Result() interface {} {
	return status.result
}

func (status *simpleTaskStatus) Status() error {
	completed := atomic.LoadUintptr(&status.completed)
	if completed == 0 {
		return NotCompleted
	}
	return status.status
}

func (status *simpleTaskStatus) Poll() {
	completed := atomic.LoadUintptr(&status.completed)
	if completed != 0 {
		return
	}
	status.cond.Wait()
}

func runTask(taskStatus *simpleTaskStatus, task func () (interface {}, error)) {
	defer func () {
		r := recover()
		if r != nil {
			taskStatus.status = &PanickedStatus { r }
		}
	}()
	taskStatus.result, taskStatus.status = task()
}

func (runner *SimpleTaskRunner) Run(task func () (interface {}, error)) (TaskStatus, error) {
	taskStatus := &simpleTaskStatus {
		condMutex: sync.Mutex {},
		completed: 0,
		status: NotCompleted,
		result: nil,
	}
	taskStatus.cond = sync.NewCond(&taskStatus.condMutex)

	go func () {
		runTask(taskStatus, task)
		atomic.StoreUintptr(&taskStatus.completed, 1)
		taskStatus.cond.Broadcast()
	}()

	return taskStatus, nil
}
