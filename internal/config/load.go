package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads and parses a single sops-config.yaml file, validating that
// every user and rule has the fields required to resolve it later.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	for i, u := range cfg.Users {
		if u.Name == "" {
			return nil, fmt.Errorf("%s: users[%d]: name is required", path, i)
		}
	}
	for i, r := range cfg.Rules {
		if r.PathRegex == "" {
			return nil, fmt.Errorf("%s: rules[%d]: path_regex is required", path, i)
		}
		if r.EncryptedRegex == "" {
			return nil, fmt.Errorf("%s: rules[%d]: encrypted_regex is required", path, i)
		}
		if r.Priority == nil {
			return nil, fmt.Errorf("%s: rules[%d]: priority is required", path, i)
		}
	}

	return &cfg, nil
}
