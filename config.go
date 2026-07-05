package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the plugin's configuration loaded from
// ~/.config/herdr/plugins/config/nicolegros.herdr-launcher/config.toml
type Config struct {
	Paths    []string `toml:"paths"`     // directories whose children become entries
	Projects []string `toml:"projects"`  // individual directories listed directly as entries
	OnCreate string   `toml:"on_create"` // shell command to run after creating a new workspace
}

// configDir returns the path to the plugin's config directory.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr", "plugins", "config", "nicolegros.herdr-launcher"), nil
}

// configPath returns the path to the plugin's config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// loadConfig reads and parses the plugin config file. If the file does not
// exist, it returns an error with guidance on where to create it.
func loadConfig() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if os.IsNotExist(err) {
			return Config{}, fmt.Errorf("config not found at %s\n\nCreate it with:\n  mkdir -p %s\n  echo 'paths = [\"~/Developer\"]' > %s", path, filepath.Dir(path), path)
		}
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.Paths) == 0 && len(cfg.Projects) == 0 {
		return Config{}, fmt.Errorf("config at %s has no paths or projects defined\n\nAdd at least one:\n  paths = [\"~/Developer\"]\n  projects = [\"~/.config\"]", path)
	}

	return cfg, nil
}

// expandPath resolves ~ and environment variables in a path.
func expandPath(path string) string {
	home, _ := os.UserHomeDir()
	if path == "~" {
		return home
	}
	if len(path) > 1 && path[:2] == "~/" {
		path = filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}
