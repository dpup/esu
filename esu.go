// Package esu provides wrappers around the AWS-SDK for locating ECS tasks. It
// can be used for basic service discovery in lieu of using ELBs infront of your
// containerized tasks.
//
// An assumption is that each task has one canonical container, for example a
// web server. For multi-container tasks, the canonical container's name should
// match the service name.
package esu
