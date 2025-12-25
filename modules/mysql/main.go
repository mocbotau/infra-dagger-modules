// MySQL module for running MySQL server as a service for testing
package main

import (
	"context"
	"fmt"

	"dagger/mysql/internal/dagger"
)

type Mysql struct {
	// +private
	Version string
	// +private
	RootPassword string
	// +private
	Database string
	// +private
	Ctr *dagger.Container
	// +private
	Svc *dagger.Service
}

func New(
	// MySQL version to use
	// +default="8.0"
	version string,
	// Root password for MySQL
	// +default="root"
	rootPassword string,
	// Database name to create
	// +default="test_db"
	database string,
) *Mysql {
	return &Mysql{
		Version:      version,
		RootPassword: rootPassword,
		Database:     database,
	}
}

// Base returns the base MySQL container
func (m *Mysql) Base() *dagger.Container {
	return dag.Container().
		From("mysql:"+m.Version).
		WithEnvVariable("MYSQL_ROOT_PASSWORD", m.RootPassword).
		WithEnvVariable("MYSQL_DATABASE", m.Database).
		WithExposedPort(3306)
}

// Service returns the MySQL service
func (m *Mysql) Service(ctx context.Context) *dagger.Service {
	if m.Svc != nil {
		return m.Svc
	}

	ctr := m.Ctr
	if ctr == nil {
		ctr = m.Base()
	}

	m.Svc = ctr.AsService()
	return m.Svc
}

// Client returns a container that can connect to the MySQL service
func (m *Mysql) Client(ctx context.Context) *dagger.Container {
	return dag.Container().
		From("mysql:"+m.Version).
		WithServiceBinding("db", m.Service(ctx)).
		WithExec([]string{
			"sh", "-c",
			"until mysqladmin ping -h db --silent; do echo 'Waiting for MySQL...'; sleep 2; done",
		})
}

// ConnectionString returns the connection string for connecting to MySQL from a bound service
func (m *Mysql) ConnectionString() string {
	return fmt.Sprintf("mysql://root:%s@db:3306/%s", m.RootPassword, m.Database)
}
