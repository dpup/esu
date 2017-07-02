// Package esu provides wrappers around the AWS-SDK for locating ECS tasks. It
// can be used for basic service discovery in lieu of using ELBs infront of your
// containerized tasks.
//
// An assumption is that each task has one canonical container, for example a
// web server. For multi-container tasks, the canonical container's name should
// match the service name.
package esu

import (
	"fmt"
	"strings"
)

// ARN contains fields within an ARN used by ECS services.
//
// arn:aws:ecs:region:account-id:task-definition/task-definition-family-name:task-definition-revision-number
// arn:aws:ecs:region:account-id:container/container-id
// arn:aws:ecr:region:account-id:repository/repo:tag
type ARN struct {
	Prefix   string // Region + Account + Resource Type
	Resource string
	Revision string
}

// ShortName returns "Resource:Revision"
func (td ARN) ShortName() string {
	if td.Revision != "" {
		return fmt.Sprintf("%s:%s", td.Resource, td.Revision)
	}
	return td.Resource
}

// String returns the full ARN (if fields are set).
func (td ARN) String() string {
	if td.Prefix == "" {
		return td.ShortName()
	}
	return fmt.Sprintf("%s/%s", td.Prefix, td.ShortName())
}

// ParseARN parses an ARN.
func ParseARN(arn string) ARN {
	var prefix string
	var resource string
	var revision string

	i := strings.Index(arn, "/")
	if i != -1 {
		prefix = arn[:i]
		arn = arn[i+1:]
	}
	j := strings.Index(arn, ":")
	if j != -1 {
		resource = arn[:j]
		revision = arn[j+1:]
	} else {
		resource = arn
	}
	return ARN{prefix, resource, revision}
}
