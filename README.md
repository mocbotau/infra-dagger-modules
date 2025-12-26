# Dagger Modules

A collection of reusable Dagger modules for CI/CD pipelines and infrastructure automation.

## Modules

### Infrastructure & Deployment

- **`blueprints/generic-deploy`** - Generic deployment blueprint for common deployment patterns
- **`modules/docker`** - Docker container operations and management
- **`modules/git-repo`** - Git repository operations including semantic versioning and tagging

### CI/CD

- **`modules/golang-ci`** - Go/Golang continuous integration workflows
- **`modules/node-ci`** - Node.js continuous integration workflows
- **`modules/python-ci`** - Python continuous integration workflows

### Services & Integrations

- **`modules/infisical`** - Infisical secrets management integration
- **`modules/mysql`** - MySQL database operations

## Usage

```bash
# Call a module from the root
dagger call -m modules/git-repo --source=. get-next-version

# Use with SSH for git operations
dagger call -m modules/git-repo --source=. --ssh=${SSH_AUTH_SOCK} tag-and-push
```

## Semantic Versioning

This monorepo uses semantic versioning for all modules together. The `git-repo` module automatically determines version bumps based on commit messages. Prepend an appropriate
versioning to change the next version bump type.

- **`[major]`** - Breaking changes → v1.0.0 → v2.0.0
- **`[patch]`** - Bug fixes → v1.0.0 → v1.0.1
- **Default (no marker)** or **`[minor]`** - New features → v1.0.0 → v1.1.0
- **`[skip]`** - Don't version

### Examples

```bash
# Commit with patch bump
git commit -m "[patch] Fix docker build cache issue"

# Commit with major bump
git commit -m "[major] Breaking: Refactor module interface"

# Commit with default minor bump
git commit -m "Add new golang-ci test features"
```

## Development

```bash
# Initialize a new module
dagger init --sdk=go modules/my-module

# Develop/update module code generation
cd modules/my-module
dagger develop
```
