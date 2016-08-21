package esu

import (
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// TaskFinder provides a wrapper around the AWS-SDK for locating ECS tasks.
type TaskFinder struct {
	cluster string
	ecs     *ecs.ECS
	ec2     *ec2.EC2
}

// NewTaskFinder returns a new task finder. It is as thread-safe as the
// underlying AWS SDK :)
func NewTaskFinder(sess *session.Session, cluster string) *TaskFinder {
	return &TaskFinder{
		cluster: cluster,
		ecs:     ecs.New(sess),
		ec2:     ec2.New(sess),
	}
}

// Services returns a list of ARNs for all services active on a cluster.
func (f *TaskFinder) Services() ([]string, error) {
	var nextToken *string
	services := []string{}
	for {
		resp, err := f.ecs.ListServices(&ecs.ListServicesInput{
			Cluster:    aws.String(f.cluster),
			MaxResults: aws.Int64(10),
			NextToken:  nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, str := range resp.ServiceArns {
			services = append(services, *str)
		}
		nextToken = resp.NextToken
		if nextToken == nil {
			return services, nil
		}
	}
}

// Tasks returns information about a service's running tasks, sorted first by
// public DNS name and then port.
func (f *TaskFinder) Tasks(service string) ([]TaskInfo, error) {
	tasksArns, err := f.fetchTasks(service)
	if err != nil {
		return nil, err
	}
	if len(tasksArns) == 0 {
		return []TaskInfo{}, nil
	}
	tasks, err := f.describeTasks(tasksArns)
	if err != nil {
		return nil, err
	}
	instances, err := f.locateTasks(tasks)
	if err != nil {
		return nil, err
	}
	infos := make([]TaskInfo, len(tasks))
	for i := range tasks {
		t := tasks[i]
		in := instances[i]
		port, err := f.getPortForTask(t, service)
		if err != nil {
			return nil, fmt.Errorf("%s, cluster=%s, service=%s, task=%s", err, f.cluster, service, *t.TaskArn)
		}
		infos[i] = TaskInfo{
			DesiredStatus:    ECSTaskStatus(realString(t.DesiredStatus)),
			LastStatus:       ECSTaskStatus(realString(t.LastStatus)),
			StartedAt:        realTime(t.StartedAt),
			StoppedAt:        realTime(t.StoppedAt),
			PublicDNSName:    realString(in.PublicDnsName),
			PrivateDNSName:   realString(in.PrivateDnsName),
			PublicIPAddress:  realString(in.PublicIpAddress),
			PrivateIPAddress: realString(in.PrivateIpAddress),
			Port:             port,
		}
	}
	sort.Sort(taskInfoList(infos))
	return infos, nil
}

// getPortForTasks looks up the containers associated with a task. For multi-
// container tasks, look for a container with the same name as the service. For
// example, if "foobaz" service runs an application container and a "mysql"
// container, for the purpose of this library the application should be named
// "foobaz". If multiple ports are mapped, the first one is taken.
func (f *TaskFinder) getPortForTask(t *ecs.Task, service string) (int, error) {
	var c *ecs.Container
	if len(t.Containers) == 0 {
		return 0, fmt.Errorf("no containers configured")
	} else if len(t.Containers) == 1 {
		c = t.Containers[0]
	} else {
		for _, cc := range t.Containers {
			if *cc.Name == service {
				c = cc
				break
			}
		}
		if c == nil {
			return 0, fmt.Errorf("ambiguous, multi-container task, one container should match service name.")
		}
	}
	if c.NetworkBindings == nil || len(c.NetworkBindings) == 0 {
		return 0, fmt.Errorf("no network bindings")
	}
	return int(*c.NetworkBindings[0].HostPort), nil
}

func (f *TaskFinder) locateTasks(tasks []*ecs.Task) ([]*ec2.Instance, error) {
	if len(tasks) == 0 {
		return []*ec2.Instance{}, nil
	}
	ciArns := make([]*string, len(tasks))
	for i, task := range tasks {
		ciArns[i] = task.ContainerInstanceArn
	}
	resp, err := f.ecs.DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		ContainerInstances: ciArns,
		Cluster:            aws.String(f.cluster),
	})
	if err != nil {
		return nil, propagate(err, "ecs describe container instances")
	}
	if len(resp.Failures) != 0 {
		// TODO: This only shows first error.
		return nil, fmt.Errorf("describe container failure on %s: %s", resp.Failures[0].Arn, resp.Failures[0].Reason)
	}
	ec2Ids := make([]*string, len(resp.ContainerInstances))
	for i, ci := range resp.ContainerInstances {
		ec2Ids[i] = ci.Ec2InstanceId
	}
	return f.locateInstances(ec2Ids)
}

func (f *TaskFinder) locateInstances(ec2Ids []*string) ([]*ec2.Instance, error) {
	resp, err := f.ec2.DescribeInstances(&ec2.DescribeInstancesInput{
		DryRun:      aws.Bool(false),
		InstanceIds: ec2Ids,
	})
	if err != nil {
		return nil, propagate(err, "ec2 describe instances")
	}
	instances := []*ec2.Instance{}
	for _, r := range resp.Reservations {
		// TODO: under what situation does this return multiple items?
		instances = append(instances, r.Instances[0])
	}
	return instances, nil
}

func (f *TaskFinder) describeTasks(tasksArns []*string) ([]*ecs.Task, error) {
	resp, err := f.ecs.DescribeTasks(&ecs.DescribeTasksInput{
		Tasks:   tasksArns,
		Cluster: aws.String(f.cluster),
	})
	if err != nil {
		return nil, propagate(err, "ecs describe tasks")
	}
	if len(resp.Failures) != 0 {
		// TODO: This only shows first error.
		return nil, fmt.Errorf("describe task failure on %s: %s", resp.Failures[0].Arn, resp.Failures[0].Reason)
	}
	// Filter out stopped tasks, we still return tasks in the process of stopping.
	tasks := []*ecs.Task{}
	for _, t := range resp.Tasks {
		if t.LastStatus != nil && ECSTaskStatus(*t.LastStatus) != ECSTaskStatusStopped {
			tasks = append(tasks, t)
		}
	}
	return tasks, nil
}

func (f *TaskFinder) fetchTasks(service string) ([]*string, error) {
	// ListTasks queries based off "DesiredState" not current state, we STOPPED as
	// well so we can see running tasks that are in the process of stopping.
	tasks, err := f.fetchTasksWithStatus(service, ECSTaskStatusRunning)
	if err != nil {
		return nil, err
	}
	stoppingTasks, err := f.fetchTasksWithStatus(service, ECSTaskStatusStopped)
	if err != nil {
		return nil, err
	}
	tasks = append(tasks, stoppingTasks...)
	return tasks, nil
}

func (f *TaskFinder) fetchTasksWithStatus(service string, desiredStatus ECSTaskStatus) ([]*string, error) {
	var nextToken *string
	tasks := []*string{}
	for {
		resp, err := f.ecs.ListTasks(&ecs.ListTasksInput{
			Cluster:       aws.String(f.cluster),
			ServiceName:   aws.String(service),
			DesiredStatus: aws.String(string(desiredStatus)),
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, propagate(err, "ecs list tasks")
		}
		for _, str := range resp.TaskArns {
			tasks = append(tasks, str)
		}
		nextToken = resp.NextToken
		if nextToken == nil {
			return tasks, nil
		}
	}
}

func realString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func realTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func propagate(err error, msg string) error {
	return fmt.Errorf("%s: %s", msg, err)
}
