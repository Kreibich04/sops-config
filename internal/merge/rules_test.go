package merge

import (
	"reflect"
	"testing"

	"github.com/Kreibich04/sops-config/internal/config"
)

func intp(i int) *int { return &i }

func TestBuildRulesPrioritySortStableTieBreak(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{
			{Name: "A", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"AABB"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "a/.*", EncryptedRegex: "^data$", Priority: intp(10), Groups: []string{"g"}},
			{PathRegex: "b/.*", EncryptedRegex: "^data$", Priority: intp(5), Groups: []string{"g"}},
			{PathRegex: "c/.*", EncryptedRegex: "^data$", Priority: intp(10), Groups: []string{"g"}},
		},
	})

	reg, _ := MergeUsers([]config.Discovered{root})
	rules, diags := BuildRules(reg, []config.Discovered{root})

	var warnings int
	for _, d := range diags {
		if d.Level == Warning {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("expected 1 duplicate-priority warning, got %d (%v)", warnings, diags)
	}

	got := make([]string, len(rules))
	for i, r := range rules {
		got[i] = r.PathRegex
	}
	want := []string{"b/.*", "a/.*", "c/.*"} // priority 5 first, then the two priority-10 rules in discovery order
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got order %v, want %v", got, want)
	}
}

func TestBuildRulesUnknownGroupIsError(t *testing.T) {
	root := disc(".", true, &config.Config{
		Rules: []config.Rule{
			{PathRegex: "a/.*", EncryptedRegex: "^data$", Priority: intp(1), Groups: []string{"nonexistent"}},
		},
	})
	reg, _ := MergeUsers([]config.Discovered{root})
	_, diags := BuildRules(reg, []config.Discovered{root})
	if !HasErrors(diags) {
		t.Fatalf("expected error for unknown group, got %v", diags)
	}
}

func TestBuildRulesEmptyKeysIsError(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{{Name: "A", Groups: []string{"g"}}}, // no pgp/age keys
		Rules: []config.Rule{
			{PathRegex: "a/.*", EncryptedRegex: "^data$", Priority: intp(1), Groups: []string{"g"}},
		},
	})
	reg, _ := MergeUsers([]config.Discovered{root})
	_, diags := BuildRules(reg, []config.Discovered{root})
	if !HasErrors(diags) {
		t.Fatalf("expected error for empty pgp/age, got %v", diags)
	}
}

func TestBuildRulesDedupeAndSortKeys(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{
			{Name: "A", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"BB", "AA"}}},
			{Name: "B", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"AA", "CC"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "a/.*", EncryptedRegex: "^data$", Priority: intp(1), Groups: []string{"g"}},
		},
	})
	reg, _ := MergeUsers([]config.Discovered{root})
	rules, diags := BuildRules(reg, []config.Discovered{root})
	if HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	want := []string{"AA", "BB", "CC"}
	if !reflect.DeepEqual(rules[0].PGP, want) {
		t.Fatalf("got %v, want %v", rules[0].PGP, want)
	}
}
