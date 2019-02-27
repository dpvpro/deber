package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"deber/pkg/constants"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"io/ioutil"
	"os"
	"time"
)

const (
	ContainerStateRunning    = "running"
	ContainerStateCreated    = "created"
	ContainerStateExited     = "exited"
	ContainerStateRestarting = "restarting"
	ContainerStatePaused     = "paused"
	ContainerStateDead       = "dead"
)

type Docker struct {
	client *client.Client
	ctx    context.Context
}

func New() (*Docker, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}

	return &Docker{
		client: cli,
		ctx:    context.Background(),
	}, nil
}

func (docker *Docker) IsImageBuilt(image string) (bool, error) {
	list, err := docker.client.ImageList(docker.ctx, types.ImageListOptions{})
	if err != nil {
		return false, err
	}

	for i := range list {
		for j := range list[i].RepoTags {
			if list[i].RepoTags[j] == image {
				return true, nil
			}
		}
	}

	return false, nil
}

func (docker *Docker) IsContainerCreated(container string) (bool, error) {
	list, err := docker.client.ContainerList(docker.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return false, err
	}

	for i := range list {
		for j := range list[i].Names {
			if list[i].Names[j] == "/"+container {
				return true, nil
			}
		}
	}

	return false, nil
}

func (docker *Docker) IsContainerStarted(container string) (bool, error) {
	list, err := docker.client.ContainerList(docker.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return false, err
	}

	for i := range list {
		for j := range list[i].Names {
			if list[i].Names[j] == "/"+container {
				if list[i].State == ContainerStateRunning {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (docker *Docker) IsContainerStopped(container string) (bool, error) {
	list, err := docker.client.ContainerList(docker.ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return false, err
	}

	for i := range list {
		for j := range list[i].Names {
			if list[i].Names[j] == "/"+container {
				if list[i].State != ContainerStateRunning {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (docker *Docker) BuildImage(name, dockerfile string) (*types.ImageBuildResponse, error) {
	buffer := new(bytes.Buffer)
	writer := tar.NewWriter(buffer)
	header := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfile)),
	}
	options := types.ImageBuildOptions{
		Tags:   []string{name},
		Remove: true,
	}

	err := writer.WriteHeader(header)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write([]byte(dockerfile))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	response, err := docker.client.ImageBuild(docker.ctx, buffer, options)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (docker *Docker) CreateContainer(name, image, buildDir, tarball string) error {
	hostArchivesDir := fmt.Sprintf("/tmp/%s", name)
	hostSourceDir := os.Getenv("PWD")
	hostBuildDir := fmt.Sprintf("%s/../%s", hostSourceDir, buildDir)
	srcTarball := fmt.Sprintf("%s/../%s", hostSourceDir, tarball)
	dstTarball := fmt.Sprintf("%s/%s", hostBuildDir, tarball)
	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: hostArchivesDir,
				Target: constants.ContainerArchivesDir,
			}, {
				Type:   mount.TypeBind,
				Source: hostSourceDir,
				Target: constants.ContainerSourceDir,
			}, {
				Type:   mount.TypeBind,
				Source: hostBuildDir,
				Target: constants.ContainerBuildDir,
			},
		},
	}
	config := &container.Config{
		Image: image,
	}

	for _, mnt := range hostConfig.Mounts {
		err := os.MkdirAll(mnt.Source, os.ModePerm)
		if err != nil {
			return err
		}
	}

	if tarball != "" {
		buffer, err := ioutil.ReadFile(srcTarball)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(dstTarball, buffer, 0664)
		if err != nil {
			return err
		}
	}

	_, err := docker.client.ContainerCreate(docker.ctx, config, hostConfig, nil, name)
	if err != nil {
		return err
	}

	return nil
}

func (docker *Docker) StartContainer(container string) error {
	options := types.ContainerStartOptions{}

	return docker.client.ContainerStart(docker.ctx, container, options)
}

func (docker *Docker) StopContainer(container string) error {
	timeout := time.Second

	return docker.client.ContainerStop(docker.ctx, container, &timeout)
}

func (docker *Docker) RemoveContainer(container string) error {
	options := types.ContainerRemoveOptions{}

	return docker.client.ContainerRemove(docker.ctx, container, options)
}

func (docker *Docker) ExecContainer(container string, cmd ...string) (*types.HijackedResponse, error) {
	config := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}

	response, err := docker.client.ContainerExecCreate(docker.ctx, container, config)
	if err != nil {
		return nil, err
	}

	hijack, err := docker.client.ContainerExecAttach(docker.ctx, response.ID, config)
	if err != nil {
		return nil, err
	}

	return &hijack, nil
}
