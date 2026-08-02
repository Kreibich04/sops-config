package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := NewRootCmd()
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestGenerateHappyPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sops-config.yaml"), `
users:
  - name: Admin
    groups: [admin]
    keys:
      pgp: [AABB11223344]
rules:
  - path_regex: .*/muc/secrets/.*
    encrypted_regex: '^(data|stringData)$'
    priority: 10
    groups: [admin]
`)

	_, stderr, err := run(t, "generate", "--root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, stderr)
	}

	out, readErr := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if readErr != nil {
		t.Fatalf("expected .sops.yaml to be written: %v", readErr)
	}
	if !strings.Contains(string(out), "pgp: AABB11223344") {
		t.Fatalf(".sops.yaml missing expected content:\n%s", out)
	}
}

func TestValidateUnknownGroupFails(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sops-config.yaml"), `
rules:
  - path_regex: .*
    encrypted_regex: '^(data)$'
    priority: 1
    groups: [typo-group]
`)

	_, stderr, err := run(t, "validate", "--root", root)
	if err == nil {
		t.Fatal("expected validate to fail on unknown group")
	}
	if !strings.Contains(stderr, "typo-group") {
		t.Fatalf("expected diagnostic to mention the offending group, got: %s", stderr)
	}

	if _, statErr := os.Stat(filepath.Join(root, ".sops.yaml")); statErr == nil {
		t.Fatal("validate must never write .sops.yaml")
	}
}

func TestValidateZeroRulesFailsUnlessForce(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sops-config.yaml"), "users: []\nrules: []\n")

	if _, _, err := run(t, "validate", "--root", root); err == nil {
		t.Fatal("expected validate to fail on zero resolved rules")
	}

	stdout, _, err := run(t, "validate", "--root", root, "--force")
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}
	if !strings.Contains(stdout, "0 rules") {
		t.Fatalf("expected stdout to mention 0 rules, got: %s", stdout)
	}
}

func TestGenerateSubdirScoping(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sops-config.yaml"), `
users:
  - name: MucAdmin
    groups: [muc-admin]
    keys:
      age: [age1mucmucmucmucmucmucmuc]
`)
	writeFile(t, filepath.Join(root, "muc", "sops-config.yaml"), `
rules:
  - path_regex: secrets/.*
    encrypted_regex: '^(data)$'
    priority: 10
    groups: [muc-admin]
`)

	_, stderr, err := run(t, "generate", "--root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, stderr)
	}

	out, _ := os.ReadFile(filepath.Join(root, ".sops.yaml"))
	if !strings.Contains(string(out), "^muc/.*secrets/.*") {
		t.Fatalf("expected scoped path_regex, got:\n%s", out)
	}
}
