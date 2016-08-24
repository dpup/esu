// Sample application that monitors the state of a service's tasks.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dpup/esu"
)

var region = flag.String("region", "us-east-1", "Which EC2 region to use")
var cluster = flag.String("cluster", "", "Cluster name to list tasks for")
var service = flag.String("service", "", "The service to monitor")

func main() {
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{
		Region: region,
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		log.Println("failed to create session:", err)
		os.Exit(1)
	}

	tm := esu.NewTaskMonitor(sess, *cluster, *service)
	tm.OnTaskChange = func(tasks []esu.TaskInfo) {
		log.Println("available tasks:")
		for _, task := range tasks {
			log.Println("  ", task)
		}
	}
	tm.OnStatusChange = func(tasks []esu.TaskInfo) {
		log.Println("status changed:")
		for _, task := range tasks {
			log.Println("  ", task)
		}
	}
	tm.OnError = func(err error) {
		log.Println("error detected:")
		log.Println("  ", err)
	}
	cancel := tm.Monitor()

	// Wait for ctrl+c to exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	cancel <- true
	log.Println("Exiting")
}
