package client

import (
	"context"
	"fmt"
	"sync"

	infisical "github.com/infisical/go-sdk"
)

var (
	clientInstance infisical.InfisicalClientInterface
	clientMutex    sync.RWMutex
)

// Config holds the configuration for creating an Infisical client
type Config struct {
	SiteURL      string
	ClientID     string
	ClientSecret string
	ProjectID    string
	Environment  string
}

// GetClient returns a cached client or creates a new one if needed
func GetClient(ctx context.Context, cfg Config) (infisical.InfisicalClientInterface, error) {
	clientMutex.RLock()
	if clientInstance != nil {
		clientMutex.RUnlock()
		return clientInstance, nil
	}
	clientMutex.RUnlock()

	clientMutex.Lock()
	defer clientMutex.Unlock()

	// Double-check after acquiring write lock
	if clientInstance != nil {
		return clientInstance, nil
	}

	client := infisical.NewInfisicalClient(ctx, infisical.Config{
		SiteUrl: cfg.SiteURL,
	})

	_, err := client.Auth().UniversalAuthLogin(cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with infisical: %w", err)
	}

	clientInstance = client
	return clientInstance, nil
}

// RetrieveSecret retrieves a secret from Infisical
func RetrieveSecret(ctx context.Context, cfg Config, key string) (string, error) {
	client, err := GetClient(ctx, cfg)
	if err != nil {
		return "", err
	}

	secret, err := client.Secrets().Retrieve(infisical.RetrieveSecretOptions{
		ProjectID:   cfg.ProjectID,
		Environment: cfg.Environment,
		SecretKey:   key,
		Type:        "shared",
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", key, err)
	}

	return secret.SecretValue, nil
}
