package esu

import (
	"fmt"
	"time"
)

// ECSTaskStatus are the different task states.
// See http://docs.aws.amazon.com/AmazonECS/latest/developerguide/task_life_cycle.html
type ECSTaskStatus string

const (
	// ECSTaskStatusRunning indicates the task is running and taking traffic.
	ECSTaskStatusRunning ECSTaskStatus = "RUNNING"
	// ECSTaskStatusPending indicates the task has been started.
	ECSTaskStatusPending ECSTaskStatus = "PENDING"
	// ECSTaskStatusStopped indicates the task has been stopped.
	ECSTaskStatusStopped ECSTaskStatus = "STOPPED"
)

// TaskInfo specifies information about a task running on ECS. A service may
// have multiple tasks associated with it.
type TaskInfo struct {
	DesiredStatus    ECSTaskStatus
	LastStatus       ECSTaskStatus
	StartedAt        time.Time
	StoppedAt        time.Time
	Port             int
	PublicDNSName    string
	PublicIPAddress  string
	PrivateDNSName   string
	PrivateIPAddress string
}

func (ti TaskInfo) String() string {
	if ti.DesiredStatus != ti.LastStatus {
		return fmt.Sprintf("[%s > %s] %s:%d / %s:%d", ti.LastStatus, ti.DesiredStatus, ti.PublicDNSName, ti.Port, ti.PrivateDNSName, ti.Port)
	}
	return fmt.Sprintf("[%s] %s:%d / %s:%d", ti.LastStatus, ti.PublicDNSName, ti.Port, ti.PrivateDNSName, ti.Port)
}

type taskInfoList []TaskInfo

func (a taskInfoList) Len() int      { return len(a) }
func (a taskInfoList) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a taskInfoList) Less(i, j int) bool {
	if a[i].PublicDNSName == a[j].PublicDNSName {
		return a[i].Port < a[j].Port
	}
	return a[i].PublicDNSName < a[j].PublicDNSName
}

func taskInfosEqual(a, b []TaskInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func runningTasks(tasks []TaskInfo) []TaskInfo {
	running := []TaskInfo{}
	for _, t := range tasks {
		if t.LastStatus == ECSTaskStatusRunning && t.DesiredStatus == ECSTaskStatusRunning {
			running = append(running, t)
		}
	}
	return running
}
