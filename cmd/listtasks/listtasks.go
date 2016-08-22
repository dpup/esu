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

var cluster = flag.String("cluster", "", "Cluster name to list tasks for")

func main() {
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
		CredentialsChainVerboseErrors: aws.Bool(true),
	})
	if err != nil {
		fmt.Println("failed to create session:", err)
		os.Exit(1)
	}

	tf := esu.NewTaskFinder(sess, *cluster)

	services, err := tf.Services()
	if err != nil {
		fmt.Println("failed to fetch services:", err)
		os.Exit(1)
	}
	for _, s := range services {
		fmt.Println(s)
		tasks, err := tf.Tasks(s)
		if err != nil {
			fmt.Println("failed to query tasks:", err)
			os.Exit(1)
		}
		for _, task := range tasks {
			fmt.Println(task)
		}
	}
}
