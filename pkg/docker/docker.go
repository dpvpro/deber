// Package docker wraps Docker Go SDK for internal usage in deber
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

const (
	// APIVersion constant is the minimum supported version of Docker Engine API
	APIVersion = "1.30"
)

// Docker struct represents Docker client.
type Docker struct {
	cli *client.Client
	ctx context.Context
}

// New function creates fresh Docker struct and connects to Docker Engine.
func New() (*Docker, error) {
	cli, err := client.NewClientWithOpts(client.WithVersion(APIVersion))
	if err != nil {
		return nil, err
	}

	fmt.Println("cli - ", cli)
	fmt.Println("docker new  - ", &Docker{ cli: cli, ctx: context.Background(), })
	return &Docker{
		cli: cli,
		ctx: context.Background(),
	}, nil
}
