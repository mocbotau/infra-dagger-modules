// A Dagger module for Git repository operations including semantic versioning
//
// This module provides functionality to automatically determine and push semantic version tags
// to Git repositories based on commit messages.
package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dagger/git-repo/internal/dagger"
)

const ghHost = "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

type GitRepo struct {
	// +private
	Ctr *dagger.Container
}

// BumpType represents the type of version bump
type BumpType string

const (
	BumpSkip  BumpType = "skip"
	BumpPatch BumpType = "patch"
	BumpMinor BumpType = "minor"
	BumpMajor BumpType = "major"
)

func New(
	// The source code directory of the Git repository
	// +defaultPath="."
	source *dagger.Directory,
	// The SSH socket for authenticating with the Git repository
	ssh *dagger.Socket,
) *GitRepo {
	ctr := dag.Container().
		From("alpine/git:latest").
		WithNewFile("/root/.ssh/known_hosts", ghHost).
		WithEnvVariable("SSH_AUTH_SOCK", "/var/ssh.sock").
		WithUnixSocket("/var/ssh.sock", ssh).
		WithExec([]string{"git", "config", "--global", "user.name", "Dagger CI"}).
		WithExec([]string{"git", "config", "--global", "user.email", "masterofcubesau@gmail.com"}).
		WithExec([]string{"git", "config", "--global", "url.ssh://git@github.com/.insteadOf", "https://github.com/"}).
		WithEnvVariable("CACHE_BUSTER", time.Now().String()).
		WithMountedDirectory("/repo", source).
		WithWorkdir("/repo")

	return &GitRepo{
		Ctr: ctr,
	}
}

// GetNextVersion determines the next semantic version from the git repository
// and returns it as a string (e.g., "v1.2.3").
// By default, it will analyse the most recent commit messages for version bump markers, for instance:
//   - [major] in commit message -> major version bump (v1.0.0 -> v2.0.0)
//   - [minor] in commit message -> minor version bump (v1.0.0 -> v1.1.0)
//   - [patch] in commit message -> patch version bump (v1.0.0 -> v1.0.1)
//   - [skip] in commit message -> no version bump (v1.0.0 -> v1.0.0)
//   - default (no marker) -> minor version bump (v1.0.0 -> v1.1.0)
func (m *GitRepo) GetNextVersion(
	ctx context.Context,
	// Optionally force a specific bump type
	// +optional
	forceBump string,
) (string, error) {
	latestTag, err := m.Ctr.
		WithExec([]string{"git", "fetch", "--tags"}).
		WithExec([]string{"git", "tag", "-l", "--sort=-version:refname"}).
		Stdout(ctx)

	if err != nil || strings.TrimSpace(latestTag) == "" {
		// No tags exist, start with v0.0.0
		latestTag = "v0.0.0"
	} else {
		// Get the first line (latest version)
		lines := strings.Split(strings.TrimSpace(latestTag), "\n")
		if len(lines) > 0 {
			latestTag = strings.TrimSpace(lines[0])
		} else {
			latestTag = "v0.0.0"
		}
	}

	major, minor, patch, err := parseVersion(latestTag)
	if err != nil {
		return "", fmt.Errorf("failed to parse version %s: %w", latestTag, err)
	}

	// Determine bump type
	bumpType := BumpMinor // default
	if forceBump != "" {
		bumpType = BumpType(forceBump)
	} else {
		commitMsg, err := m.Ctr.
			WithExec([]string{"git", "log", "HEAD", "--pretty=format:%s", "-1"}).
			Stdout(ctx)

		if err == nil {
			bumpType = determineBumpType(commitMsg)
		}
	}

	if bumpType == BumpSkip {
		// No version bump
		return "", ErrVersionBumpSkipped
	}

	switch bumpType {
	case BumpMajor:
		major++
		minor = 0
		patch = 0
	case BumpMinor:
		minor++
		patch = 0
	case BumpPatch:
		patch++
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch), nil
}

// TagAndPush creates a new semantic version tag and pushes it to the remote repository
// Returns the version tag that was created and pushed
func (m *GitRepo) TagAndPush(
	ctx context.Context,
	// New version to tag, otherwise determined automatically
	// +optional
	version string,
	// Optionally force a specific bump type if version is not provided
	// +optional
	forceBump string,
	// Optional release message for the tag
	// +optional
	message string,
) (string, error) {
	// Determine version if not provided
	if version == "" {
		var err error
		version, err = m.GetNextVersion(ctx, forceBump)
		if err == ErrVersionBumpSkipped {
			return "", nil // No tag created
		}

		if err != nil {
			return "", fmt.Errorf("failed to determine next version: %w", err)
		}
	}

	if message == "" {
		message = fmt.Sprintf("Release %s", version)
	}

	// Create and push the tag in a single pipeline
	_, err := m.Ctr.
		WithExec([]string{"git", "tag", "-a", version, "-m", message}).
		WithExec([]string{"git", "push", "origin", version}).
		Sync(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to create and push tag: %w", err)
	}

	return version, nil
}

// parseVersion parses a semantic version string (e.g., "v1.2.3") into its components
func parseVersion(version string) (major, minor, patch int, err error) {
	version = strings.TrimPrefix(version, "v")

	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)

	if len(matches) != 4 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", version)
	}

	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	patch, _ = strconv.Atoi(matches[3])

	return major, minor, patch, nil
}

// determineBumpType analyses commit messages to determine the appropriate version bump
func determineBumpType(commitMessages string) BumpType {
	lowerMessages := strings.ToLower(commitMessages)

	if strings.Contains(lowerMessages, "[skip]") {
		return BumpSkip
	}

	if strings.Contains(lowerMessages, "[major]") {
		return BumpMajor
	}

	if strings.Contains(lowerMessages, "[minor]") {
		return BumpMinor
	}

	if strings.Contains(lowerMessages, "[patch]") {
		return BumpPatch
	}

	// Default to minor bump
	return BumpMinor
}
