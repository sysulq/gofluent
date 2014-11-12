package task

import (
	"testing"
	"time"
)

type DummyTaskStatus struct {
	result interface {}
	error error
}

func (status *DummyTaskStatus) Status() error { return status.error }

func (status *DummyTaskStatus) Result() interface {} { return status.result }

func (*DummyTaskStatus) Poll() {}

type DummyTaskRunner struct {
	lastRunTask func () (interface {}, error)
}

func (runner *DummyTaskRunner) Run(task func () (interface {}, error)) (TaskStatus, error) {
	retval, err := task()
	return &DummyTaskStatus { retval, err }, nil
}

type recurringTaskArgs struct {
	id int64
	firedOn time.Time
	spec RecurringTaskSpec
}

func TestGoal(t *testing.T) {
	var now time.Time
	runner := &DummyTaskRunner {}
	sched := NewRecurringTaskScheduler(func () time.Time { return now }, runner)
	results := make([]recurringTaskArgs, 0, 10)

	now = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	id1, err := sched.RegisterTask(
		RecurringTaskSpec {
			month: nil,
			dayOfWeek: nil,
			dayOfMonth: nil,
			hour: []int { 0, 1 },
			minute: []int { 10, 20 },
			rightAt: time.Time {},
		},
		func (id int64, on time.Time, spec *RecurringTaskSpec) (interface {}, error) {
			results = append(results, recurringTaskArgs { id, on, *spec })
			t.Logf("%d: %v, %v", id, on, spec)
			return nil, nil
		},
	)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	sched.ProcessEvent()

	id2, err := sched.RegisterTask(
		RecurringTaskSpec {
			month: nil,
			dayOfWeek: nil,
			dayOfMonth: nil,
			hour: []int { 0 },
			minute: []int { 0, 10 },
			rightAt: time.Time {},
		},
		func (id int64, on time.Time, spec *RecurringTaskSpec) (interface {}, error) {
			results = append(results, recurringTaskArgs { id, on, *spec })
			t.Logf("%d: %v, %v", id, on, spec)
			return nil, nil
		},
	)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	sched.ProcessEvent()

	if id1 == id2 {
		t.Fail()
	}

	go sched.ProcessEvent()
	diff, _, err := sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }
	go sched.ProcessEvent()

	t.Logf("results=%d", len(results))
	if len(results) != 1 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 10 * 60 * 1000000000 { t.Fail() }

	// 
	now = time.Date(1970, 1, 1, 0, 10, 0, 0, time.UTC)

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }

	t.Logf("results=%d", len(results))
	if len(results) != 2 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }

	t.Logf("results=%d", len(results))
	if len(results) != 3 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 10 * 60 * 1000000000 { t.Fail() }

	//
	now = time.Date(1970, 1, 1, 0, 20, 0, 0, time.UTC)

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }

	t.Logf("results=%d", len(results))
	if len(results) != 4 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 50 * 60 * 1000000000 { t.Fail() }

	//
	now = time.Date(1970, 1, 1, 1, 10, 0, 0, time.UTC)

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }

	t.Logf("results=%d", len(results))
	if len(results) != 5 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 10 * 60 * 1000000000 { t.Fail() }

	//
	now = time.Date(1970, 1, 1, 1, 20, 0, 0, time.UTC)

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	if diff != 0 { t.Fail() }

	t.Logf("results=%d", len(results))
	if len(results) != 6 { t.Fail() }
	go sched.ProcessEvent() // for update

	go sched.ProcessEvent()
	diff, _, err = sched.RunNext()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	t.Logf("diff=%d", diff)
	// nanoseconds from 1970/1/1 01:20:00 to 1970/1/2 00:00:00
	if diff != (22 * 60 + 40) * 60 * 1000000000 { t.Fail() }
}
