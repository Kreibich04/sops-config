package merge

import "github.com/Kreibich04/sops-config/internal/config"

// Result is the outcome of running the merge pipeline over a directory tree.
type Result struct {
	Rules []ResolvedRule
}

// Run discovers every sops-config.yaml under root, merges users, and
// resolves rules into a final, priority-sorted rule set. It is the single
// entrypoint shared by the generate and validate commands, so they can never
// drift in what they check.
func Run(root string) (Result, []Diagnostic, error) {
	discovered, err := config.Discover(root)
	if err != nil {
		return Result{}, nil, err
	}

	reg, userDiags := MergeUsers(discovered)
	rules, ruleDiags := BuildRules(reg, discovered)

	diags := append(userDiags, ruleDiags...)
	return Result{Rules: rules}, diags, nil
}
