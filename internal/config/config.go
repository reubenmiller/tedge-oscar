package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type RegistryCredential struct {
	Registry string `toml:"registry" json:"registry" yaml:"registry"`
	Username string `toml:"username" json:"username" yaml:"username"`
	Password string `toml:"password" json:"password" yaml:"password"`
}

type Config struct {
	ImageDir   string               `toml:"image_dir" json:"image_dir" yaml:"image_dir"`
	Registries []RegistryCredential `toml:"registries" json:"registries" yaml:"registries"`
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./tedge-oscar.toml"
	}
	return filepath.Join(home, ".config", "tedge-oscar", "config.toml")
}

func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}

func (c *Config) Expand() {
	c.ImageDir = expandEnvVars(c.ImageDir)
	for i := range c.Registries {
		c.Registries[i].Registry = expandEnvVars(c.Registries[i].Registry)
		c.Registries[i].Username = expandEnvVars(c.Registries[i].Username)
		c.Registries[i].Password = expandEnvVars(c.Registries[i].Password)
	}
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	cfg.Expand()
	return &cfg, nil
}

func (c *Config) FindCredential(registry string) *RegistryCredential {
	for _, cred := range c.Registries {
		if cred.Registry == registry {
			return &cred
		}
	}
	return nil
}
