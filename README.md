# ESU (ECS Service Utility)

_From the department of nested acronyms, comes a set of utilities for locating
and monitoring ECS services._

**Status: Experimental**

This repository contains a Go library that wraps the AWS SDK and offers
utilities for querying and monitoring ECS tasks. The original purpose was to
facilitate service discovery in lieu of an ELB in front of each service, but
there are many other purposes.

To list or locate tasks one off, use `TaskFinder`:

```go
tf := esu.NewTaskFinder(sess, *cluster)
tasks, err := tf.Tasks("website")
```

To monitor the status of a service, use `TaskMonitor`:

```go
tm := esu.NewTaskMonitor(sess, "sites", "website")
tm.OnTaskChange = func(tasks []esu.TaskInfo) { ... }
tm.OnError = func(err error) { ... }
tm.Monitor()
```

The data return about a Task, aggregated from several API calls is represented
by the `TaskInfo` struct:

```go
type TaskInfo struct {
  TaskDefinition   string
  DesiredStatus    ECSTaskStatus  // RUNNING, PENDING, STOPPED
  LastStatus       ECSTaskStatus
  StartedAt        time.Time
  Port             int
  PublicDNSName    string
  PublicIPAddress  string
  PrivateDNSName   string
  PrivateIPAddress string
}
```

Full API docs can be found here: https://godoc.org/github.com/dpup/esu

## Tools

_[List Tasks](./cmd/listtasks/listtasks.go)_ - Show all services and tasks
running on an ECS cluster

    go run cmd/listtasks/listtasks.go --cluster=sites

_[Monitor Service](./cmd/monitor/monitor.go)_ - Monitors status of a service,
printing out status changes to runing tasks.

    go run cmd/monitor/monitor.go --cluster=sites --service=website

_[Update Task](./cmd/updatetask/updatetask.go)_ - Updates the container image of
a service and waits for the tasks to be updated.

    go run cmd/updatetask/updatetask.go --world=prod --service=website --tag=build-38

(This task assumes the cluster is named `prod-cluster` and the task definition
for the servies is `prod-website`.)

(Make sure credentials are available in the environment)

## Contributing

Questions, comments, bug reports, and pull requests are all welcome. Submit them
[on the project issue tracker](https://github.com/dpup/gohubbub/esu/new).

## License

Copyright 2016 [Daniel Pupius](http://pupius.co.uk). Licensed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
