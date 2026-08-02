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

func TestMergeUsersCrossFileGroupUsage(t *testing.T) {
	root := disc(".", true, &config.Config{})
	sub := disc("muc", false, &config.Config{Users: []config.User{
		{Name: "MucAdmin", Groups: []string{"muc-admin"}, Keys: config.Keys{Age: []string{"age1"}}},
	}})

	reg, diags := MergeUsers([]config.Discovered{root, sub})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
	users := reg.UsersInGroup("muc-admin")
	if len(users) != 1 || users[0].Name != "MucAdmin" {
		t.Fatalf("expected MucAdmin resolvable globally, got %v", users)
	}
}
