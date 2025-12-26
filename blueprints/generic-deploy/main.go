package main

import (
	"context"

	"dagger/generic-deploy/internal/dagger"
)

type GenericDeploy struct {
	// Source code directory
	// +private
	Source *dagger.Directory
	// +private
	InfisicalClientSecret *dagger.Secret
}

func New(
	// Source code directory
	// +defaultPath="."
	source *dagger.Directory,
	infisicalClientSecret *dagger.Secret,
) *GenericDeploy {
	return &GenericDeploy{
		Source:                source,
		InfisicalClientSecret: infisicalClientSecret,
	}
}

// BuildAndPush builds and pushes the Docker image to the container registry
func (m *GenericDeploy) BuildAndPush(
	ctx context.Context,
	// Environment to build image for
	// +default="staging"
	env string,
	repoName string,
	// Additional build arguments, format KEY=VALUE
	// +optional
	buildArgs []string,
) (string, error) {
	docker := dag.Docker(m.Source, m.InfisicalClientSecret, repoName, dagger.DockerOpts{
		Environment: env,
	})

	return docker.Build(dagger.DockerBuildOpts{
		BuildArgs: buildArgs,
	}).Publish(ctx)
}
