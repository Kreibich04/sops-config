package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadFileValid(t *testing.T) {
	path := writeTemp(t, `
users:
  - name: "Admin"
    groups: [admin]
    keys:
      pgp: ["AABB11223344"]
rules:
  - path_regex: secrets/.*
    encrypted_regex: '^(data)$'
    priority: 10
    groups: [admin]
`)
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Users) != 1 || cfg.Users[0].Name != "Admin" {
		t.Fatalf("unexpected users: %+v", cfg.Users)
	}
	if len(cfg.Rules) != 1 || *cfg.Rules[0].Priority != 10 {
		t.Fatalf("unexpected rules: %+v", cfg.Rules)
	}
}

func TestLoadFileMalformedYAML(t *testing.T) {
	path := writeTemp(t, "users: [\n  - name: broken\n")
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadFileUnknownFieldIsError(t *testing.T) {
	path := writeTemp(t, `
users:
  - name: "Admin"
    comnent: "typo of comment"
    keys:
      pgp: ["AABB11223344"]
`)
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestLoadFileRejectsInvalidUsersAndRules(t *testing.T) {
	cases := map[string]string{
		"empty group name": `
users:
  - name: A
    groups: [""]
    keys:
      pgp: ["AABB11223344"]
`,
		"duplicate group": `
users:
  - name: A
    groups: [admin, admin]
    keys:
      pgp: ["AABB11223344"]
`,
		"user with no keys": `
users:
  - name: A
    groups: [admin]
`,
		"malformed pgp key": `
users:
  - name: A
    keys:
      pgp: ["not-hex"]
`,
		"malformed age key": `
users:
  - name: A
    keys:
      age: ["not-an-age-key"]
`,
		"duplicate key": `
users:
  - name: A
    keys:
      pgp: ["AABB11223344", "AABB11223344"]
`,
		"rule with no groups": `
rules:
  - path_regex: .*
    encrypted_regex: '^(data)$'
    priority: 1
    groups: []
`,
	}
	for name, contents := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTemp(t, contents)
			if _, err := LoadFile(path); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLoadFileMissingFields(t *testing.T) {
	cases := map[string]string{
		"missing user name": `
users:
  - groups: [admin]
`,
		"missing path_regex": `
rules:
  - encrypted_regex: '^(data)$'
    priority: 1
`,
		"missing encrypted_regex": `
rules:
  - path_regex: .*
    priority: 1
`,
		"missing priority": `
rules:
  - path_regex: .*
    encrypted_regex: '^(data)$'
`,
	}
	for name, contents := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeTemp(t, contents)
			if _, err := LoadFile(path); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
