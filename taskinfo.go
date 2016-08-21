package esu

import (
	"fmt"
	"time"
)

// TaskInfo specifies information about a task running on ECS. A service may
// have multiple tasks associated with it.
type TaskInfo struct {
	DesiredStatus    string
	LastStatus       string
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
