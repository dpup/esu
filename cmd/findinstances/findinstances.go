// Sample application that lists all services and tasks running on a cluster.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dpup/esu"
)

var region = flag.String("region", "us-east-1", "Which EC2 region to use")
var cluster = flag.String("cluster", "", "Cluster name to list tasks for")
var service = flag.String("service", "", "Service to fetch instances for")

func main() {
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{
		Region: region,
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		fmt.Println("failed to create session:", err)
		os.Exit(1)
	}

	tf := esu.NewTaskFinder(sess, *cluster)
	tasks, err := tf.Tasks(*service)
	if err != nil {
		fmt.Println("failed to query tasks:", err)
		os.Exit(1)
	}
	for _, task := range tasks {
		fmt.Println(task.PublicIPAddress)
	}
}
