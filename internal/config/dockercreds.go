package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ...existing code...

type dockerConfig struct {
	Auths map[string]struct {
		Auth     string `json:"auth"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"auths"`
}

func LoadDockerCredentials(registry string) (username, password string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	paths := []string{
		filepath.Join(home, ".docker", "config.json"),
		filepath.Join(home, ".config", "docker", "config.json"),
	}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()
		var cfg dockerConfig
		if err := json.NewDecoder(f).Decode(&cfg); err != nil {
			continue
		}
		for reg, entry := range cfg.Auths {
			if strings.Contains(registry, reg) || strings.Contains(reg, registry) {
				if entry.Username != "" && entry.Password != "" {
					return entry.Username, entry.Password, nil
				}
				if entry.Auth != "" {
					decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
					if err == nil {
						parts := strings.SplitN(string(decoded), ":", 2)
						if len(parts) == 2 {
							return parts[0], parts[1], nil
						}
					}
				}
			}
		}
	}
	return "", "", fmt.Errorf("no docker credentials found for registry: %s", registry)
}
