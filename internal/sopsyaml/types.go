// Package sopsyaml renders resolved rules into the .sops.yaml format
// consumed by Mozilla SOPS.
package sopsyaml

type sopsFile struct {
	CreationRules []creationRule `yaml:"creation_rules"`
}

type creationRule struct {
	PathRegex      string `yaml:"path_regex"`
	EncryptedRegex string `yaml:"encrypted_regex,omitempty"`
	Comment        string `yaml:"comment,omitempty"`
	PGP            string `yaml:"pgp,omitempty"`
	Age            string `yaml:"age,omitempty"`
}
