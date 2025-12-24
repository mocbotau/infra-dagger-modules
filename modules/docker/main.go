package main

import (
	"context"
	"fmt"

	"dagger/docker/internal/dagger"
)

const (
	registryRepo = "cloud"
)

type Docker struct {
	// +private
	Container *dagger.Container
	// +private
	Environment string
	// +private
	InfisicalClientSecret *dagger.Secret
	// +private
	RepoName string
	// +private
	Source *dagger.Directory
}

func New(
	// The source code directory containing the Dockerfile
	source *dagger.Directory,
	// The Infisical client secret for retrieving Docker Hub credentials
	infisicalClientSecret *dagger.Secret,
	// The repository name for the Docker image
	repoName string,
	// The environment to tag the Docker image with
	// +default="staging"
	environment string,
) *Docker {
	return &Docker{
		Environment:           environment,
		InfisicalClientSecret: infisicalClientSecret,
		RepoName:              repoName,
		Source:                source,
	}
}

// Build builds the Dockerfile present in the source directory
func (m *Docker) Build(ctx context.Context) *Docker {
	m.Container = m.Source.DockerBuild()
	return m
}

// BuildContainer builds the passed in Docker container
func (m *Docker) BuildContainer(ctx context.Context, container *dagger.Container) *Docker {
	m.Container = container
	return m
}

// GetContainer returns the built Docker container
func (m *Docker) GetContainer(ctx context.Context) (*dagger.Container, error) {
	return m.Container.Sync(ctx)
}

// Publish builds and pushes the container image to Docker Hub
func (m *Docker) Publish(ctx context.Context) (string, error) {
	if m.Container == nil {
		return "", fmt.Errorf("container is not built yet")
	}

	if m.RepoName == "" {
		return "", fmt.Errorf("repository name is not set")
	}

	infisical := dag.Infisical(m.InfisicalClientSecret, m.Environment)

	username := infisical.GetSecret("DOCKERHUB_USERNAME")
	password := infisical.GetSecret("DOCKERHUB_PASSWORD")

	usernameString, err := username.Plaintext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get dockerhub username: %w", err)
	}

	imageTag := fmt.Sprintf("%s/%s:%s-%s", usernameString, registryRepo, m.RepoName, m.Environment)

	address, err := m.Container.
		WithRegistryAuth("docker.io", usernameString, password).
		Publish(ctx, imageTag)

	if err != nil {
		return "", fmt.Errorf("failed to publish image: %w", err)
	}

	return address, nil
}
