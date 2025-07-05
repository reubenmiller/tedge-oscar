package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	ImageDir string `toml:"image_dir" json:"image_dir" yaml:"image_dir"`
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./tedge-oscar.toml"
	}
	return filepath.Join(home, ".config", "tedge-oscar", "config.toml")
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
