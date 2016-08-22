package esu

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

// DefaultPollFreq specifies how often to check ECS for task changes, under
// normal operation.
const DefaultPollFreq = time.Second * 10

// DefaultVolatilePollFreq specifies how often to check ECS for task changes,
// when a task has been noted as being in a pending state or soon to be stopped.
const DefaultVolatilePollFreq = time.Second * 1

// numUpdatesForStable specifies how many successful updates should have been
// seen before the monitor is considered stable, following an error.
const numUpdatesForStable = 5

// TaskMonitor is
type TaskMonitor struct {
	Service          string
	PollFreq         time.Duration
	VolatilePollFreq time.Duration
	OnStatusChange   func([]TaskInfo)
	OnTaskChange     func([]TaskInfo)
	OnError          func(error)
	taskFinder       *TaskFinder
	allTasks         []TaskInfo
	runningTasks     []TaskInfo
	updatesSinceErr  int
}

// NewTaskMonitor returns a new task monitor.
func NewTaskMonitor(sess *session.Session, cluster string, service string) *TaskMonitor {
	return &TaskMonitor{
		Service:          service,
		PollFreq:         DefaultPollFreq,
		VolatilePollFreq: DefaultVolatilePollFreq,
		taskFinder:       NewTaskFinder(sess, cluster),
	}
}

// RunningTasks returns a list of currently running tasks.
func (tm *TaskMonitor) RunningTasks() []TaskInfo {
	return tm.runningTasks
}

// AllTasks returns a list of all tasks, including pending and stopped.
func (tm *TaskMonitor) AllTasks() []TaskInfo {
	return tm.allTasks
}

// Monitor polls for changes in running tasks, relevant callbacks are executed
// when changes are detected. There should only be one active Monitor per
// instance.
func (tm *TaskMonitor) Monitor() chan<- bool {
	tm.Update()
	cancel := make(chan bool, 1)
	go func() {
		for {
			freq := tm.PollFreq
			if tm.IsVolatile() {
				freq = tm.VolatilePollFreq
			}
			select {
			case <-cancel:
				return
			case <-time.After(freq):
				tm.Update()
			}
		}
	}()
	return cancel
}

// IsVolatile returns true if any tasks have a desired status that doesn't match
// last status, or an error was encountered recently.
func (tm *TaskMonitor) IsVolatile() bool {
	if tm.updatesSinceErr < numUpdatesForStable {
		// Haven't had enough successful updates to be considered stable.
		return false
	}
	for _, t := range tm.allTasks {
		if t.DesiredStatus != t.LastStatus {
			return true
		}
	}
	return false
}

// Update queries ECS for the latest tasks and returns true if there were any
// changes in the number of running tasks.
func (tm *TaskMonitor) Update() bool {
	tasks, err := tm.taskFinder.Tasks(tm.Service)
	if err != nil {
		tm.updatesSinceErr = 0
		if tm.OnError != nil {
			tm.OnError(err)
		}
		return false
	}
	tm.updatesSinceErr++
	if !taskInfosEqual(tasks, tm.allTasks) {
		tm.allTasks = tasks
		if tm.OnStatusChange != nil {
			tm.OnStatusChange(tasks)
		}
		running := runningTasks(tasks)
		if !taskInfosEqual(running, tm.runningTasks) {
			tm.runningTasks = running
			if tm.OnTaskChange != nil {
				tm.OnTaskChange(running)
			}
			return true
		}
	}
	return false
}
