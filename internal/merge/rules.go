package merge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Kreibich04/sops-config/internal/config"
)

// ResolvedRule is a config.Rule after path-scoping and group-to-key
// resolution, ready to be rendered as a SOPS creation_rule.
type ResolvedRule struct {
	PathRegex      string
	EncryptedRegex string
	Comment        string
	Priority       int
	PGP            []string
	Age            []string
}

// BuildRules resolves every rule from every discovered config against the
// user registry, in discovery order (root's rules first, then subdirectories
// in walk order, in-file order preserved within each file). That ordering is
// preserved by the later stable priority sort, so equal-priority rules keep
// a deterministic root-then-subdir order.
func BuildRules(reg *UserRegistry, discovered []config.Discovered) ([]ResolvedRule, []Diagnostic) {
	var rules []ResolvedRule
	var diags []Diagnostic
	seenPriority := make(map[int]string)

	for _, d := range discovered {
		source := sourcePath(d)
		for i, r := range d.Config.Rules {
			label := fmt.Sprintf("rules[%d]", i)

			if !strings.HasPrefix(r.PathRegex, "^") {
				where := "over the whole repo"
				if d.Dir != "." {
					where = "anywhere under " + d.Dir + "/, not just directly inside it"
				}
				diags = append(diags, Diagnostic{Warning, source, fmt.Sprintf(
					"%s: path_regex %q is unanchored; it's matched as a substring search %s (consider a leading \"^\" to anchor it)",
					label, r.PathRegex, where,
				)})
			}

			scoped := ScopePathRegex(d.Dir, r.PathRegex)
			if _, err := regexp.Compile(scoped); err != nil {
				diags = append(diags, Diagnostic{Error, source, fmt.Sprintf("%s: invalid path_regex %q: %v", label, scoped, err)})
				continue
			}
			if _, err := regexp.Compile(r.EncryptedRegex); err != nil {
				diags = append(diags, Diagnostic{Error, source, fmt.Sprintf("%s: invalid encrypted_regex %q: %v", label, r.EncryptedRegex, err)})
				continue
			}

			pgpSet := map[string]struct{}{}
			ageSet := map[string]struct{}{}
			badRule := false
			for _, g := range r.Groups {
				users := reg.UsersInGroup(g, d.Dir)
				if len(users) == 0 {
					diags = append(diags, Diagnostic{Error, source, fmt.Sprintf("%s: group %q has no matching users", label, g)})
					badRule = true
					continue
				}
				for _, u := range users {
					for _, k := range u.PGP {
						pgpSet[k] = struct{}{}
					}
					for _, k := range u.Age {
						ageSet[k] = struct{}{}
					}
				}
			}
			if badRule {
				continue
			}

			pgp := sortedKeys(pgpSet)
			age := sortedKeys(ageSet)
			if len(pgp) == 0 && len(age) == 0 {
				diags = append(diags, Diagnostic{Error, source, fmt.Sprintf("%s: resolves to no PGP or Age keys", label)})
				continue
			}

			priority := *r.Priority
			if prev, ok := seenPriority[priority]; ok {
				diags = append(diags, Diagnostic{Warning, source, fmt.Sprintf("%s: priority %d duplicates a rule in %s", label, priority, prev)})
			} else {
				seenPriority[priority] = fmt.Sprintf("%s: %s", source, label)
			}

			rules = append(rules, ResolvedRule{
				PathRegex:      scoped,
				EncryptedRegex: r.EncryptedRegex,
				Comment:        r.Comment,
				Priority:       priority,
				PGP:            pgp,
				Age:            age,
			})
		}
	}

	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })

	return rules, diags
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
