package config

import (
	"context"
	"fmt"

	"oras.land/oras-go/v2/registry/remote/credentials"
)

// LoadDockerCredentials retrieves Docker credentials for the given registry.
func LoadDockerCredentials(registry string) (username, password string, err error) {
	credStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return "", "", err
	}
	cred, err := credStore.Get(context.Background(), registry)
	if err != nil {
		return "", "", err
	}
	if cred.Username != "" && cred.Password != "" {
		return cred.Username, cred.Password, nil
	}
	return "", "", fmt.Errorf("no docker credentials found for registry: %s", registry)
}
