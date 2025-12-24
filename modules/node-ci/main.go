package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/node-ci/internal/dagger"
)

// NodeCi module for Node.js CI tasks
type NodeCi struct {
	// +private
	NodeVersion string
	// +private
	PackageManager PackageManager
	// +private
	Source *dagger.Directory
	// +private
	Ctr *dagger.Container
}

type PackageManager string

const (
	NPM  PackageManager = "npm"
	Yarn PackageManager = "yarn"
	PNPM PackageManager = "pnpm"
)

func New(
	// The source code directory
	// +ignore=["**/node_modules"]
	source *dagger.Directory,
	// The Node version to use
	// +default="20"
	nodeVersion string,
	// The package manager to use (npm, yarn, pnpm)
	// +default="npm"
	packageManager PackageManager,
) *NodeCi {
	return &NodeCi{
		NodeVersion:    nodeVersion,
		PackageManager: packageManager,
		Source:         source,
		Ctr:            nil,
	}
}

// Base returns the base Node container
func (m *NodeCi) Base() *dagger.Container {
	container := dag.Container().
		From("node:" + m.NodeVersion + "-alpine").
		WithExec([]string{"apk", "add", "--no-cache", "git"})

	switch m.PackageManager {
	case PNPM:
		container = container.WithExec([]string{"npm", "install", "-g", "pnpm"})
	case Yarn:
		container = container.WithExec([]string{"npm", "install", "-g", "yarn"})
	}

	return container
}

// getPackageManagerCache returns the appropriate cache path and volume name
func (m *NodeCi) getPackageManagerCache() (string, string) {
	switch m.PackageManager {
	case NPM:
		return "/root/.npm", "npm-cache"
	case Yarn:
		return "/usr/local/share/.cache/yarn", "yarn-cache"
	case PNPM:
		return "/root/.local/share/pnpm/store", "pnpm-cache"
	default:
		return "/root/.npm", "npm-cache"
	}
}

// getLockfile returns the lockfile name for the package manager
func (m *NodeCi) getLockfile() string {
	switch m.PackageManager {
	case NPM:
		return "package-lock.json"
	case Yarn:
		return "yarn.lock"
	case PNPM:
		return "pnpm-lock.yaml"
	default:
		return "package-lock.json"
	}
}

// getInstallCommand returns the install command for the package manager
func (m *NodeCi) getInstallCommand() []string {
	switch m.PackageManager {
	case NPM:
		return []string{"npm", "ci"}
	case Yarn:
		return []string{"yarn", "install", "--frozen-lockfile"}
	case PNPM:
		return []string{"pnpm", "install", "--frozen-lockfile"}
	default:
		return []string{"npm", "ci"}
	}
}

// getContainer returns the container, installing dependencies if needed
func (m *NodeCi) getContainer(ctx context.Context) *dagger.Container {
	if m.Ctr != nil {
		return m.Ctr
	}
	return m.Install(ctx).Ctr
}

// Install installs dependencies with caching and returns the NodeCi instance for chaining
func (m *NodeCi) Install(ctx context.Context) *NodeCi {
	cachePath, volumeName := m.getPackageManagerCache()
	lockfile := m.getLockfile()

	container := m.Base().
		WithWorkdir("/app").
		WithMountedCache(cachePath, dag.CacheVolume(volumeName)).
		WithFile("/app/package.json", m.Source.File("package.json"))

	lockfileEntry, err := m.Source.File(lockfile).ID(ctx)
	if err == nil && lockfileEntry != "" {
		container = container.WithFile("/app/"+lockfile, m.Source.File(lockfile))
	}

	m.Ctr = container.WithExec(m.getInstallCommand()).WithDirectory("/app", m.Source)

	return m
}

// WithExec runs a command and returns the NodeCi instance for chaining. Prepends package manager run
func (m *NodeCi) WithExec(
	ctx context.Context,
	// Command to run (e.g., "lint", "test", "prettier")
	cmd string,
	// Additional arguments
	// +optional
	args []string,
) *NodeCi {
	cmdParts := []string{string(m.PackageManager), "run", cmd}
	if len(args) > 0 {
		cmdParts = append(cmdParts, args...)
	}

	m.Ctr = m.getContainer(ctx).WithExec(cmdParts)
	return m
}

// Exec runs a command and returns the output immediately. Prepends package manager run
func (m *NodeCi) Exec(
	ctx context.Context,
	// Command to run (e.g., "lint", "test", "prettier")
	cmd string,
	// Additional arguments
	// +optional
	args []string,
) (string, error) {
	return m.WithExec(ctx, cmd, args).Stdout(ctx)
}

// Lint runs the lint command and returns output
func (m *NodeCi) Lint(ctx context.Context) (string, error) {
	return m.Exec(ctx, "lint", nil)
}

// WithLint runs the lint command for chaining
func (m *NodeCi) WithLint(ctx context.Context) *NodeCi {
	return m.WithExec(ctx, "lint", nil)
}

// Test runs the test command and returns output
func (m *NodeCi) Test(ctx context.Context) (string, error) {
	return m.Exec(ctx, "test", nil)
}

// WithTest runs the test command for chaining
func (m *NodeCi) WithTest(ctx context.Context) *NodeCi {
	return m.WithExec(ctx, "test", nil)
}

// WithBuild builds the application with optional Next.js cache
func (m *NodeCi) WithBuild(
	ctx context.Context,
	// Use Next.js build cache
	// +optional
	useNextCache bool,
	// Additional environment variables for the build in KEY=VALUE format
	// +optional
	buildEnv []string,
) *NodeCi {
	container := m.getContainer(ctx)

	for _, env := range buildEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			container = container.WithEnvVariable(parts[0], parts[1])
		}
	}

	if useNextCache {
		container = container.WithMountedCache("/app/.next/cache", dag.CacheVolume("nextjs-cache"))
	}

	m.Ctr = container.WithExec([]string{string(m.PackageManager), "run", "build"})
	return m
}

// Build builds the application and returns the container
func (m *NodeCi) Build(
	ctx context.Context,
	// Use Next.js build cache
	// +optional
	useNextCache bool,
	// Additional environment variables for the build in KEY=VALUE format
	// +optional
	buildEnv []string,
) *dagger.Container {
	return m.WithBuild(ctx, useNextCache, buildEnv).Ctr
}

// BuildOutput returns the build output directory
func (m *NodeCi) BuildOutput(
	ctx context.Context,
	// Use Next.js build cache
	// +optional
	useNextCache bool,
	// Additional environment variables for the build in KEY=VALUE format
	// +optional
	buildEnv []string,
	// Output directory path
	// +default=".next"
	outputPath string,
) *dagger.Directory {
	return m.WithBuild(ctx, useNextCache, buildEnv).Directory(outputPath)
}

// Directory returns a directory from the container
func (m *NodeCi) Directory(
	// Directory path relative to /app
	path string,
) *dagger.Directory {
	return m.Ctr.Directory("/app/" + path)
}

// Container returns the underlying container
func (m *NodeCi) Container(ctx context.Context) *dagger.Container {
	return m.getContainer(ctx)
}

// Stdout returns the stdout of the last executed command
func (m *NodeCi) Stdout(ctx context.Context) (string, error) {
	if m.Ctr == nil {
		return "", fmt.Errorf("no commands executed yet")
	}
	return m.Ctr.Stdout(ctx)
}

// Stderr returns the stderr of the last executed command
func (m *NodeCi) Stderr(ctx context.Context) (string, error) {
	if m.Ctr == nil {
		return "", fmt.Errorf("no commands executed yet")
	}
	return m.Ctr.Stderr(ctx)
}

// Sync executes the pipeline and returns success
func (m *NodeCi) Sync(ctx context.Context) (bool, error) {
	if m.Ctr == nil {
		return false, fmt.Errorf("no commands executed yet")
	}
	_, err := m.Ctr.Sync(ctx)
	return err == nil, err
}

// WithCommand runs an arbitrary command and returns NodeCi for chaining
func (m *NodeCi) WithCommand(
	ctx context.Context,
	// Command to run as a string (will be split on spaces)
	command string,
) *NodeCi {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return m
	}

	m.Ctr = m.getContainer(ctx).WithExec(parts)
	return m
}

// RunCommand runs an arbitrary command and returns the output
func (m *NodeCi) RunCommand(
	ctx context.Context,
	// Command to run as a string (will be split on spaces)
	command string,
) (string, error) {
	return m.WithCommand(ctx, command).Stdout(ctx)
}
