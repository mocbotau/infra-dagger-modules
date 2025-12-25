package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/golang-ci/internal/dagger"

	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
)

// GolangCi module for Golang CI tasks
type GolangCi struct {
	// +private
	GoVersion string
	// +private
	Source *dagger.Directory
}

func New(
	ctx context.Context,
	// The source code directory
	source *dagger.Directory,
) (*GolangCi, error) {
	goVersion, err := goVersion(ctx, source)
	if err != nil {
		return nil, err
	}

	return &GolangCi{
		GoVersion: goVersion,
		Source:    source,
	}, nil
}

// base returns a Go container with the specified variant, dependencies installed, and source code
func (m *GolangCi) base(variant string) *dagger.Container {
	return dag.Container().
		From("golang:"+m.GoVersion+"-"+variant).
		WithWorkdir("/src").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("go-mod-cache")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("go-build-cache")).
		WithFile("go.mod", m.Source.File("go.mod")).
		WithFile("go.sum", m.Source.File("go.sum")).
		WithExec([]string{"go", "mod", "download"}).
		WithDirectory("/src", m.Source)
}

// BaseAlpine returns the base alpine Go container with dependencies installed + source code
func (m *GolangCi) BaseAlpine(ctx context.Context) *dagger.Container {
	return m.base("alpine")
}

// BaseDebian returns the base debian Go container with dependencies installed + source code
func (m *GolangCi) BaseDebian(ctx context.Context) *dagger.Container {
	return m.base("trixie")
}

// Lint runs golangci-lint on the source code
func (m *GolangCi) Lint(
	ctx context.Context,
	// Go linter version
	// +default="v2.4.0"
	version string,
) (string, error) {
	return m.BaseAlpine(ctx).
		WithMountedCache("/root/.cache/golangci-lint", dag.CacheVolume("golangci-lint-cache")).
		WithExec([]string{"sh", "-c", "wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s " + version}).
		WithExec([]string{"./bin/golangci-lint", "run", "./..."}).
		Stdout(ctx)
}

// Build compiles the Go application
func (m *GolangCi) Build(ctx context.Context) (string, error) {
	return m.BaseAlpine(ctx).
		WithExec([]string{"go", "build", "./..."}).
		Stdout(ctx)
}

// Test runs Go tests with coverage
func (m *GolangCi) Test(ctx context.Context) (string, error) {
	return m.BaseDebian(ctx).
		WithExec([]string{"go", "test", "./..."}).
		Stdout(ctx)
}

// All runs lint, build, and test in parallel
func (m *GolangCi) All(
	ctx context.Context,
	// Go linter version
	// +default="v2.4.0"
	version string,
) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		_, err := m.Lint(ctx, version)
		return err
	})

	g.Go(func() error {
		_, err := m.Build(ctx)
		return err
	})

	g.Go(func() error {
		_, err := m.Test(ctx)
		return err
	})

	return g.Wait()
}

// GolangVersion returns the Go version used in the module
func (m *GolangCi) GolangVersion(ctx context.Context) string {
	return m.GoVersion
}

// goVersion extracts the major.minor Go version from go.mod
func goVersion(ctx context.Context, source *dagger.Directory) (string, error) {
	goMod, err := source.File("go.mod").Contents(ctx)
	if err != nil {
		return "", err
	}

	f, err := modfile.Parse("go.mod", []byte(goMod), nil)
	if err != nil {
		return "", err
	}

	if f.Go != nil {
		// split off the patch version if present
		var version string

		parts := strings.Split(f.Go.Version, ".")

		if len(parts) >= 2 {
			version = parts[0] + "." + parts[1]
		} else {
			version = f.Go.Version
		}

		return version, nil
	}

	return "", fmt.Errorf("go version not found in go.mod")
}
