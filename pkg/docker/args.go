package docker

import "github.com/docker/docker/api/types/mount"

// ImageBuildArgs struct represents arguments
// passed to ImageBuild().
type ImageBuildArgs struct {
	// Full parent image name,
	// placed in Dockerfile's FROM directive
	//
	// Example: ubuntu:bionic
	From string
	// Full to-be-built image name
	//
	// Example: deber:unstable
	Name string
}

// ContainerCreateArgs struct represents arguments
// passed to ContainerCreate().
type ContainerCreateArgs struct {
	Mounts []mount.Mount
	Image  string
	Name   string
	User   string
}

// ContainerExecArgs struct represents arguments
// passed to ContainerExec().
type ContainerExecArgs struct {
	Interactive bool
	Name        string
	Cmd         string
	WorkDir     string
	AsRoot      bool
	Skip        bool
	Network     bool
}
