package main

import (
	"context"

	"dagger/python-ci/internal/dagger"
)

// PythonCi module for Python CI tasks
type PythonCi struct {
	// +private
	PythonVersion string
	// +private
	Source *dagger.Directory
}

func New(
	// The source code directory
	source *dagger.Directory,
	// The Python version to use
	// +default="3.14"
	pythonVersion string,
) *PythonCi {
	return &PythonCi{
		PythonVersion: pythonVersion,
		Source:        source,
	}
}

// Base returns the base Python container
func (m *PythonCi) Base() *dagger.Container {
	return dag.Container().
		From("python:" + m.PythonVersion + "-slim")
}

// Lint runs flake8 linting on the Python source code
func (m *PythonCi) Lint(ctx context.Context) (string, error) {
	return m.Base().
		WithMountedCache(
			"/root/.cache/pip",
			dag.CacheVolume("pip-cache"),
		).
		WithExec([]string{"pip", "install", "--upgrade", "pip"}).
		WithExec([]string{"pip", "install", "flake8==7.0.0"}).
		WithDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithExec([]string{"flake8", "."}).
		Stdout(ctx)
}
