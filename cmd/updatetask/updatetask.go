// Command which updates a task definition and waits for the service to become
// stable. If the service doesn't become stable within the timeout it reverts to
// the initial task definition.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/dpup/esu"
)

var (
	region  = flag.String("region", "us-east-1", "Which EC2 region to use")
	cluster = flag.String("cluster", "", "Cluster which the service belongs to")
	service = flag.String("service", "", "The service to update")
	tag     = flag.String("tag", "", "Tag of the new image to deploy")

	timeout = flag.Duration("timeout", 2*time.Minute, "How long to wait for the task to deploy before reverting")
	force   = flag.Bool("force", false, "Whether to update the task definition regardless of current state. No rollback on failure.")

	errTimeout = errors.New("Timeout")
)

// TODO:
// - Consider moving querying of task definitions into TaskFinder.
// - Verify timeout/rollback.
// - Try to cleanup tag vs. revision vs. task definition.

func main() {
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{
		Region: region,
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		log.Fatalln("failed to create session:", err)
	}

	tf := esu.NewTaskFinder(sess, *cluster)
	tasks, err := tf.Tasks(*service)
	if err != nil {
		log.Fatalln("failed to query tasks:", err)
	}

	svc := ecs.New(sess)

	defs, err := loadTaskDefinitions(svc, tasks)
	if err != nil {
		log.Fatalln("failed to query Task Definition:", err)
	}

	if checkTag(defs, *tag) {
		log.Println("All tasks are up to date")
		return
	}

	if !isStable(defs) {
		log.Println("Tasks aren't stable. Multiple revisions active:")
		for _, task := range tasks {
			log.Println("  ", task)
		}
		if !*force {
			os.Exit(1)
			return
		}
	}

	log.Printf("Update required")

	// Use most recent task definition as a template for the service update.
	template := getLatestTaskDef(defs)
	log.Printf("Using %s:%d as template", *template.Family, *template.Revision)

	newTaskDef, err := updateTaskDef(svc, template, *tag)
	if err != nil {
		log.Fatalln("Failed to update task definition:", err)
		return
	}

	// Try to update the service.
	if err := updateService(tf, svc, newTaskDef, *cluster, *service, *timeout); err != nil {
		if err == errTimeout {
			oldTaskDef := esu.ParseARN(*template.TaskDefinitionArn).ShortName()
			log.Printf("Rolling back to %s", oldTaskDef)
			if err := updateService(tf, svc, oldTaskDef, *cluster, *service, *timeout); err != nil {
				log.Println("Error rolling back", err)
			}
		}
		log.Println("Failure :(")
		os.Exit(1)
	}

	log.Println("Success!")
}

func updateService(tf *esu.TaskFinder, svc *ecs.ECS,
	taskDef, cluster, service string,
	timeout time.Duration) error {

	resp, err := svc.UpdateService(&ecs.UpdateServiceInput{
		Cluster:        aws.String(cluster),
		Service:        aws.String(service),
		TaskDefinition: aws.String(taskDef),
	})
	if err != nil {
		return fmt.Errorf("failed to update service: %s", err)
	}

	log.Println("Service updated to:", *resp.Service.TaskDefinition)

	start := time.Now()
	for {
		time.Sleep(5 * time.Second)

		tasks, err := tf.Tasks(service)
		if err != nil {
			// Ignore error while waiting for update.
			log.Println("Failed to query tasks:", err)
			continue
		}

		wait := time.Since(start)
		for _, task := range tasks {
			log.Println("  ", task)
		}

		if checkTask(tasks, taskDef) {
			log.Printf("All tasks updated!! (%.fs)", wait.Seconds())

			return nil

		} else if wait > timeout {
			log.Println("Timedout waiting for tasks to update.")
			return errTimeout

		} else {
			log.Printf("Waiting... (%.fs)", wait.Seconds())
		}
	}
}

func updateTaskDef(svc *ecs.ECS, template *ecs.TaskDefinition, tag string) (string, error) {
	containerDef := template.ContainerDefinitions[0]
	imageARN := esu.ParseARN(*containerDef.Image)
	imageARN.Revision = tag
	containerDef.Image = aws.String(imageARN.String())
	resp, err := svc.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{containerDef},
		TaskRoleArn:          template.TaskRoleArn,
		Family:               service,
	})
	if err != nil {
		return "", err
	}
	return esu.ParseARN(*resp.TaskDefinition.TaskDefinitionArn).ShortName(), nil
}

func loadTaskDefinitions(svc *ecs.ECS, tasks []esu.TaskInfo) ([]*ecs.TaskDefinition, error) {
	defs := make([]*ecs.TaskDefinition, len(tasks))
	for i, t := range tasks {
		resp, err := svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: aws.String(t.TaskDefinition),
		})
		if err != nil {
			return nil, err
		}
		defs[i] = resp.TaskDefinition
	}
	return defs, nil
}

func getLatestTaskDef(defs []*ecs.TaskDefinition) *ecs.TaskDefinition {
	var latest *ecs.TaskDefinition
	for _, d := range defs {
		if latest == nil || *d.Revision > *latest.Revision {
			latest = d
		}
	}
	return latest
}

// checkTask returns true if all tasks belong to the given taskDef (family:revision).
func checkTask(tasks []esu.TaskInfo, taskDef string) bool {
	for _, t := range tasks {
		if t.TaskDefinition != taskDef {
			return false
		}
	}
	return true
}

// checkTag returns true if all tasks are using the provided tagged image.
func checkTag(defs []*ecs.TaskDefinition, tag string) bool {
	for i, d := range defs {
		if len(d.ContainerDefinitions) != 1 {
			log.Fatalln("Multi-container tasks are not currently supported!")
		}
		t := esu.ParseARN(*d.ContainerDefinitions[0].Image)
		log.Printf("Task %d running %s", i, t.ShortName())
		if t.Revision != tag {
			return false
		}
	}
	return true
}

// isStable returns true if all task definitions are for the same revision.
func isStable(defs []*ecs.TaskDefinition) bool {
	var r int64
	for _, d := range defs {
		if r == 0 {
			r = *d.Revision
		} else if r != *d.Revision {
			return false
		}
	}
	return true
}
