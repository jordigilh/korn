package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/images"
	ptypes "github.com/containers/podman/v5/pkg/domain/entities/types"
)

var (
	linux       = "linux"
	amd64       = "amd64"
	forceRemove = true
	quietPull   = true
)

func GetImageData(imagePullSpec string) (*ptypes.ImageInspectReport, error) {
	dockerHostEnv, ok := os.LookupEnv("DOCKER_HOST")
	if !ok {
		return nil, fmt.Errorf("environment variable DOCKER_HOST not defined in environment")
	}
	conn, err := bindings.NewConnection(context.Background(), dockerHostEnv)
	if err != nil {
		return nil, err
	}
	// Pull the image to be inspected
	id, err := images.Pull(conn, imagePullSpec, &images.PullOptions{OS: &linux, Arch: &amd64, Quiet: &quietPull})
	if err != nil {
		return nil, err
	}
	defer images.Remove(conn, id, &images.RemoveOptions{Force: &forceRemove})
	data, err := images.GetImage(conn, id[0], new(images.GetOptions).WithSize(true))
	if err != nil {
		return nil, err
	}
	return data, nil
}
