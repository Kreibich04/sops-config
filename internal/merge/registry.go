package merge

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Kreibich04/sops-config/internal/config"
)

// ResolvedUser is a User merged across every sops-config.yaml it was
// identically declared in.
type ResolvedUser struct {
	Name   string
	Groups map[string]struct{}
	PGP    []string
	Age    []string
	Source string

	// Dirs is the set of directories (relative to the discovery root, "."
	// for the root itself) this user was declared in. A user is visible to
	// a rule only if one of these directories is an ancestor of (or equal
	// to) the rule's own directory — see UserRegistry.UsersInGroup.
	Dirs map[string]struct{}
}

// UserRegistry is the set of users and group memberships merged across
// every discovered config file, along with the directory scope each user
// was declared in.
type UserRegistry struct {
	byName map[string]*ResolvedUser
}

// UsersInGroup returns every user who is a member of the given group and
// visible from fromDir, in a stable (name-sorted) order. A user is visible
// from fromDir if it was declared in fromDir itself, an ancestor of
// fromDir, or (for the root, ".") anywhere: this is what keeps a
// subdirectory's users from being able to affect rules outside its own
// subtree, while still letting root-declared users grant access everywhere.
func (r *UserRegistry) UsersInGroup(group, fromDir string) []*ResolvedUser {
	var out []*ResolvedUser
	for _, u := range r.byName {
		if _, ok := u.Groups[group]; !ok {
			continue
		}
		if !visibleFrom(u.Dirs, fromDir) {
			continue
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func visibleFrom(declaredDirs map[string]struct{}, fromDir string) bool {
	for dir := range declaredDirs {
		if dir == "." || dir == fromDir || strings.HasPrefix(fromDir, dir+"/") {
			return true
		}
	}
	return false
}

// MergeUsers combines the users declared across every discovered config file
// into a single registry. A user redeclared with identical groups/keys is a
// silent no-op that extends the user's visibility to the redeclaring
// directory too; a user redeclared with conflicting data is an error, since
// key material conflicts must never be resolved by silently picking one
// side.
func MergeUsers(discovered []config.Discovered) (*UserRegistry, []Diagnostic) {
	reg := &UserRegistry{byName: make(map[string]*ResolvedUser)}
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
				Dirs:   map[string]struct{}{d.Dir: {}},
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
				continue
			}

			existing.Dirs[d.Dir] = struct{}{}
		}
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
