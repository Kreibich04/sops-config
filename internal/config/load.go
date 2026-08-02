package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ageKeyPattern and pgpKeyPattern are sanity checks, not full format
// validators: they catch empty/placeholder/obviously-wrong values (the
// typo case) without hard-coding exact bech32 or fingerprint lengths that
// could reject a legitimately-formatted key.
var (
	ageKeyPattern = regexp.MustCompile(`^age1[a-z0-9]{20,}$`)
	pgpKeyPattern = regexp.MustCompile(`^(?:[0-9A-Fa-f]{2}){4,20}$`)
)

// LoadFile reads and parses a single sops-config.yaml file, validating that
// every user and rule has the fields required to resolve it later.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	for i, u := range cfg.Users {
		if u.Name == "" {
			return nil, fmt.Errorf("%s: users[%d]: name is required", path, i)
		}
		if err := validateGroupNames(u.Groups); err != nil {
			return nil, fmt.Errorf("%s: users[%d]: %w", path, i, err)
		}
		if err := validateKeys(u.Keys); err != nil {
			return nil, fmt.Errorf("%s: users[%d]: %w", path, i, err)
		}
	}
	for i, r := range cfg.Rules {
		if r.PathRegex == "" {
			return nil, fmt.Errorf("%s: rules[%d]: path_regex is required", path, i)
		}
		if r.Priority == nil {
			return nil, fmt.Errorf("%s: rules[%d]: priority is required", path, i)
		}
		if len(r.Groups) == 0 {
			return nil, fmt.Errorf("%s: rules[%d]: at least one group is required", path, i)
		}
		if err := validateGroupNames(r.Groups); err != nil {
			return nil, fmt.Errorf("%s: rules[%d]: %w", path, i, err)
		}
	}

	return &cfg, nil
}

func validateGroupNames(groups []string) error {
	seen := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		if g == "" {
			return fmt.Errorf("group name must not be empty")
		}
		if _, dup := seen[g]; dup {
			return fmt.Errorf("duplicate group %q", g)
		}
		seen[g] = struct{}{}
	}
	return nil
}

func validateKeys(k Keys) error {
	if len(k.PGP) == 0 && len(k.Age) == 0 {
		return fmt.Errorf("must declare at least one pgp or age key")
	}

	seen := make(map[string]struct{}, len(k.PGP)+len(k.Age))
	for _, key := range k.PGP {
		if !pgpKeyPattern.MatchString(key) {
			return fmt.Errorf("malformed pgp key %q: expected an even-length hex fingerprint or key ID", key)
		}
		if _, dup := seen["pgp:"+key]; dup {
			return fmt.Errorf("duplicate pgp key %q", key)
		}
		seen["pgp:"+key] = struct{}{}
	}
	for _, key := range k.Age {
		if !ageKeyPattern.MatchString(key) {
			return fmt.Errorf("malformed age key %q: expected an age1... recipient", key)
		}
		if _, dup := seen["age:"+key]; dup {
			return fmt.Errorf("duplicate age key %q", key)
		}
		seen["age:"+key] = struct{}{}
	}
	return nil
}
