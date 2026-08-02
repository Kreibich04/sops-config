package merge

import (
	"fmt"
	"sort"

	"github.com/Kreibich04/sops-config/internal/config"
)

// ResolvedUser is a User merged across every sops-config.yaml it was
// declared in.
type ResolvedUser struct {
	Name   string
	Groups map[string]struct{}
	PGP    []string
	Age    []string
	Source string
}

// UserRegistry is the global set of users and group memberships, merged
// across every discovered config file.
type UserRegistry struct {
	byName  map[string]*ResolvedUser
	byGroup map[string][]*ResolvedUser
}

// UsersInGroup returns every user who is a member of the given group, in a
// stable (name-sorted) order.
func (r *UserRegistry) UsersInGroup(group string) []*ResolvedUser {
	return r.byGroup[group]
}

// MergeUsers combines the users declared across every discovered config file
// into a single registry. A user redeclared with identical groups/keys is a
// silent no-op; a user redeclared with conflicting data is an error, since
// key material conflicts must never be resolved by silently picking one
// side.
func MergeUsers(discovered []config.Discovered) (*UserRegistry, []Diagnostic) {
	reg := &UserRegistry{
		byName:  make(map[string]*ResolvedUser),
		byGroup: make(map[string][]*ResolvedUser),
	}
	var diags []Diagnostic

	for _, d := range discovered {
		source := sourcePath(d)
		for _, u := range d.Config.Users {
			candidate := &ResolvedUser{
				Name:   u.Name,
				Groups: toSet(u.Groups),
				PGP:    append([]string(nil), u.Keys.PGP...),
				Age:    append([]string(nil), u.Keys.Age...),
				Source: source,
			}

			existing, ok := reg.byName[u.Name]
			if !ok {
				reg.byName[u.Name] = candidate
				continue
			}

			if !setEqual(existing.Groups, candidate.Groups) ||
				!sliceEqualUnordered(existing.PGP, candidate.PGP) ||
				!sliceEqualUnordered(existing.Age, candidate.Age) {
				diags = append(diags, Diagnostic{
					Level:  Error,
					Source: source,
					Message: fmt.Sprintf(
						"user %q conflicts with the definition in %s (differing groups or keys)",
						u.Name, existing.Source,
					),
				})
			}
		}
	}

	for _, u := range reg.byName {
		for g := range u.Groups {
			reg.byGroup[g] = append(reg.byGroup[g], u)
		}
	}
	for g, users := range reg.byGroup {
		sort.Slice(users, func(i, j int) bool { return users[i].Name < users[j].Name })
		reg.byGroup[g] = users
	}

	return reg, diags
}

func sourcePath(d config.Discovered) string {
	if d.IsRoot {
		return config.FileName
	}
	return d.Dir + "/" + config.FileName
}

func toSet(items []string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, i := range items {
		set[i] = struct{}{}
	}
	return set
}

func setEqual(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func sliceEqualUnordered(a, b []string) bool {
	return setEqual(toSet(a), toSet(b))
}
