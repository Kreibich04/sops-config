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
			{PathRegex: "^a/.*", EncryptedRegex: "^data$", Priority: intp(10), Groups: []string{"g"}},
			{PathRegex: "^b/.*", EncryptedRegex: "^data$", Priority: intp(5), Groups: []string{"g"}},
			{PathRegex: "^c/.*", EncryptedRegex: "^data$", Priority: intp(10), Groups: []string{"g"}},
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
	want := []string{"^b/.*", "^a/.*", "^c/.*"} // priority 5 first, then the two priority-10 rules in discovery order
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

func TestBuildRulesChildUserCannotEscalateToParentRule(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{
			{Name: "RootAdmin", Groups: []string{"admin"}, Keys: config.Keys{Age: []string{"age1root"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "production/.*", EncryptedRegex: ".*", Priority: intp(10), Groups: []string{"admin"}},
		},
	})
	attacker := disc("test/application", false, &config.Config{
		Users: []config.User{
			{Name: "Attacker", Groups: []string{"admin"}, Keys: config.Keys{Age: []string{"age1attacker"}}},
		},
	})

	reg, _ := MergeUsers([]config.Discovered{root, attacker})
	rules, diags := BuildRules(reg, []config.Discovered{root, attacker})
	if HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	want := []string{"age1root"}
	if !reflect.DeepEqual(rules[0].Age, want) {
		t.Fatalf("child-declared user leaked into root rule: got %v, want %v", rules[0].Age, want)
	}
}

func TestBuildRulesChildAnchorComposition(t *testing.T) {
	root := disc(".", true, &config.Config{})
	sub := disc("muc", false, &config.Config{
		Users: []config.User{
			{Name: "A", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"AABB1122"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "^secrets/.*", EncryptedRegex: "^data$", Priority: intp(1), Groups: []string{"g"}, Comment: "anchored"},
			{PathRegex: "secrets/.*", EncryptedRegex: "^data$", Priority: intp(2), Groups: []string{"g"}, Comment: "unanchored"},
		},
	})

	reg, _ := MergeUsers([]config.Discovered{root, sub})
	rules, diags := BuildRules(reg, []config.Discovered{root, sub})
	if HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].PathRegex != "^muc/secrets/.*" {
		t.Fatalf("anchored child pattern: got %q, want %q", rules[0].PathRegex, "^muc/secrets/.*")
	}
	if rules[1].PathRegex != "^muc/.*secrets/.*" {
		t.Fatalf("unanchored child pattern: got %q, want %q", rules[1].PathRegex, "^muc/.*secrets/.*")
	}
}

func TestBuildRulesWarnsOnUnanchoredRegex(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{
			{Name: "A", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"AABB1122"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "secrets/.*", EncryptedRegex: "^data$", Priority: intp(1), Groups: []string{"g"}},
		},
	})
	sub := disc("muc", false, &config.Config{
		Rules: []config.Rule{
			{PathRegex: "^secrets/.*", EncryptedRegex: "^data$", Priority: intp(2), Groups: []string{"g"}},
		},
	})

	reg, _ := MergeUsers([]config.Discovered{root, sub})
	_, diags := BuildRules(reg, []config.Discovered{root, sub})
	var warnings int
	for _, d := range diags {
		if d.Level == Warning {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("expected 1 unanchored warning (only from the root rule), got %d (%v)", warnings, diags)
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

func TestBuildRulesWholeFileEncryptedRegexOptional(t *testing.T) {
	root := disc(".", true, &config.Config{
		Users: []config.User{
			{Name: "A", Groups: []string{"g"}, Keys: config.Keys{PGP: []string{"AABB"}}},
		},
		Rules: []config.Rule{
			{PathRegex: "^secrets/.*", Priority: intp(1), Groups: []string{"g"}},
		},
	})
	reg, _ := MergeUsers([]config.Discovered{root})
	rules, diags := BuildRules(reg, []config.Discovered{root})
	if HasErrors(diags) {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if got := rules[0].EncryptedRegex; got != "" {
		t.Fatalf("EncryptedRegex = %q, want empty (whole-file encryption)", got)
	}
}
