package merge

import (
	"testing"

	"github.com/Kreibich04/sops-config/internal/config"
)

func disc(dir string, isRoot bool, cfg *config.Config) config.Discovered {
	return config.Discovered{Config: cfg, Dir: dir, IsRoot: isRoot}
}

func TestMergeUsersIdenticalDuplicateIsNoOp(t *testing.T) {
	u := config.User{Name: "Admin", Groups: []string{"admin"}, Keys: config.Keys{PGP: []string{"AABB"}}}
	root := disc(".", true, &config.Config{Users: []config.User{u}})
	sub := disc("muc", false, &config.Config{Users: []config.User{u}})

	reg, diags := MergeUsers([]config.Discovered{root, sub})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
	if len(reg.byName) != 1 {
		t.Fatalf("expected 1 merged user, got %d", len(reg.byName))
	}
}

func TestMergeUsersConflictingDuplicateIsError(t *testing.T) {
	root := disc(".", true, &config.Config{Users: []config.User{
		{Name: "Admin", Groups: []string{"admin"}, Keys: config.Keys{PGP: []string{"AABB"}}},
	}})
	sub := disc("muc", false, &config.Config{Users: []config.User{
		{Name: "Admin", Groups: []string{"admin"}, Keys: config.Keys{PGP: []string{"CCDD"}}},
	}})

	_, diags := MergeUsers([]config.Discovered{root, sub})
	if !HasErrors(diags) {
		t.Fatalf("expected an error diagnostic, got %v", diags)
	}
}

func TestMergeUsersVisibleWithinOwnSubtree(t *testing.T) {
	root := disc(".", true, &config.Config{})
	sub := disc("muc", false, &config.Config{Users: []config.User{
		{Name: "MucAdmin", Groups: []string{"muc-admin"}, Keys: config.Keys{Age: []string{"age1"}}},
	}})

	reg, diags := MergeUsers([]config.Discovered{root, sub})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}

	users := reg.UsersInGroup("muc-admin", "muc")
	if len(users) != 1 || users[0].Name != "MucAdmin" {
		t.Fatalf("expected MucAdmin resolvable from its own dir, got %v", users)
	}

	users = reg.UsersInGroup("muc-admin", "muc/prod")
	if len(users) != 1 || users[0].Name != "MucAdmin" {
		t.Fatalf("expected MucAdmin resolvable from a descendant dir, got %v", users)
	}
}

func TestMergeUsersNotVisibleOutsideDeclaringSubtree(t *testing.T) {
	root := disc(".", true, &config.Config{})
	sub := disc("test/application", false, &config.Config{Users: []config.User{
		{Name: "Attacker", Groups: []string{"admin"}, Keys: config.Keys{Age: []string{"age1attacker"}}},
	}})

	reg, diags := MergeUsers([]config.Discovered{root, sub})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}

	if users := reg.UsersInGroup("admin", "."); len(users) != 0 {
		t.Fatalf("expected a subdir user to be invisible from root, got %v", users)
	}
	if users := reg.UsersInGroup("admin", "production"); len(users) != 0 {
		t.Fatalf("expected a subdir user to be invisible from an unrelated sibling, got %v", users)
	}
}

func TestMergeUsersRootVisibleEverywhere(t *testing.T) {
	root := disc(".", true, &config.Config{Users: []config.User{
		{Name: "RootAdmin", Groups: []string{"admin"}, Keys: config.Keys{Age: []string{"age1root"}}},
	}})
	sub := disc("muc", false, &config.Config{})

	reg, _ := MergeUsers([]config.Discovered{root, sub})
	if users := reg.UsersInGroup("admin", "muc/prod"); len(users) != 1 || users[0].Name != "RootAdmin" {
		t.Fatalf("expected root user visible from any subdir, got %v", users)
	}
}
