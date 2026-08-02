package merge

import (
	"regexp"
	"strings"
)

// ScopePathRegex confines an author-written path_regex to the subtree it was
// declared in. Root-level rules (dir == ".") are returned unmodified, since
// they are meant to describe paths across the whole tree. Subdirectory rules
// have the directory prepended, quoted so directory names containing regex
// metacharacters (e.g. "my.app") are matched literally, and the whole
// pattern is anchored at the start.
//
// SOPS matches path_regex as an unanchored substring search, so without the
// leading "^" a directory named e.g. "muc" appearing anywhere else in the
// tree could accidentally satisfy a rule meant only for muc/. No trailing
// "$" is added: the author decides whether the rule should cover one file or
// an entire subtree.
func ScopePathRegex(dir, pattern string) string {
	if dir == "." {
		return pattern
	}
	pattern = strings.TrimPrefix(pattern, "/")
	prefix := regexp.QuoteMeta(dir) + "/"
	return "^" + prefix + pattern
}
