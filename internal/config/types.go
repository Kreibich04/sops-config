// Package config defines and loads the sops-config.yaml file format.
package config

// Config is the parsed contents of a single sops-config.yaml file.
type Config struct {
	Users []User `yaml:"users"`
	Rules []Rule `yaml:"rules"`
}

// User defines a person and the keys used to grant them decrypt access via
// their group memberships.
type User struct {
	Name   string   `yaml:"name"`
	Groups []string `yaml:"groups"`
	Keys   Keys     `yaml:"keys"`
}

// Keys holds the PGP fingerprints and Age recipient keys owned by a User.
type Keys struct {
	PGP []string `yaml:"pgp"`
	Age []string `yaml:"age"`
}

// Rule describes one SOPS creation_rule to generate, in terms of the groups
// that should be able to decrypt matching paths.
type Rule struct {
	PathRegex string `yaml:"path_regex"`
	// EncryptedRegex is optional; omitting it encrypts the entire file, same
	// as omitting it from a vanilla SOPS creation_rule.
	EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
	Comment        string `yaml:"comment,omitempty"`
	// Priority is a pointer so a missing value can be distinguished from an
	// explicit priority of 0.
	Priority *int     `yaml:"priority"`
	Groups   []string `yaml:"groups"`
}
