package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dpup/esu"
)

var cluster = flag.String("cluster", "", "Cluster name to list tasks for")
var service = flag.String("service", "", "The service to monitor")

func main() {
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		log.Println("failed to create session:", err)
		return
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
		log.Println("error :( ", err)
	}
	cancel := tm.Monitor()

	// Wait for ctrl+c to exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	cancel <- true
	log.Println("Exiting")
}
