package main

import (
	"context"
	"fmt"

	"dagger/infisical/internal/client"
	"dagger/infisical/internal/dagger"
)

const (
	infisicalSite = "https://infisical.masterofcubesau.com"
)

// Infisical module for managing secrets with Infisical
type Infisical struct {
	// +private
	ClientID string
	// +private
	ClientSecret string
	// +private
	Environment string
	// +private
	ProjectId string
}

func New(
	ctx context.Context,
	// The Infisical Client ID
	// +default="858472ca-5b71-4b9c-a7fd-4c9a971e5758"
	clientId string,
	// The Infisical Client Secret
	clientSecret *dagger.Secret,
	// The Infisical Project ID. Defaults to the "infrastructure" project
	// +default="85c36879-bb62-4cd6-a0c3-9eaae22f61d5"
	projectId string,
	// The environment to fetch secrets from
	environment string,
) (*Infisical, error) {
	secret, err := clientSecret.Plaintext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get infisical secret: %w", err)
	}

	// Initialize the client to verify credentials
	cfg := client.Config{
		SiteURL:      infisicalSite,
		ClientID:     clientId,
		ClientSecret: secret,
		ProjectID:    projectId,
		Environment:  environment,
	}

	_, err = client.GetClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Infisical{
		ClientID:     clientId,
		ClientSecret: secret,
		ProjectId:    projectId,
		Environment:  environment,
	}, nil
}

// WithProjectID updates the Infisical Project ID
func (m *Infisical) WithProjectID(projectId string) *Infisical {
	m.ProjectId = projectId
	return m
}

// WithEnvironment updates the Infisical environment
func (m *Infisical) WithEnvironment(environment string) *Infisical {
	m.Environment = environment
	return m
}

// GetSecret retrieves a single secret from Infisical
func (m *Infisical) GetSecret(ctx context.Context, key string) (*dagger.Secret, error) {
	cfg := client.Config{
		SiteURL:      infisicalSite,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		ProjectID:    m.ProjectId,
		Environment:  m.Environment,
	}

	secretValue, err := client.RetrieveSecret(ctx, cfg, key)
	if err != nil {
		return nil, err
	}

	return dag.SetSecret(key, secretValue), nil
}
