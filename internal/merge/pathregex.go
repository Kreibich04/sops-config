package merge

import (
	"regexp"
	"strings"
)

// ScopePathRegex confines an author-written path_regex to the subtree it was
// declared in, while keeping the fragment's own matching semantics the same
// as if the author had run it directly against paths inside their own
// directory. Root-level rules (dir == ".") are returned unmodified.
//
// SOPS matches path_regex as an unanchored substring search over the whole
// repo, so a subdirectory fragment gets the same treatment scoped to its
// own subtree:
//
//   - No leading "^": the fragment is an unanchored search *within* the
//     directory, matching regardless of how deep it occurs — e.g. dir "muc"
//     with pattern "secrets/.*" becomes "^muc/.*secrets/.*", matching both
//     "muc/secrets/x" and "muc/nested/secrets/x". This is what a reader
//     would expect from "secrets/.*" if they mentally ran it as a root
//     pattern scoped to just their own directory.
//   - A leading "^": the author is anchoring explicitly, so it's honored as
//     "immediately under my directory" — dir "muc" with pattern
//     "^secrets/.*" becomes "^muc/secrets/.*", matching "muc/secrets/x" but
//     not "muc/nested/secrets/x". The caret is stripped rather than kept
//     in place, since composing it literally ("^muc/^secrets/.*") would
//     produce a mid-string anchor that can never match.
//
// In both cases the directory component is escaped with regexp.QuoteMeta
// so directory names containing regex metacharacters (e.g. "my.app") are
// matched literally, and the whole pattern is anchored at the start: without
// the leading "^" a directory named e.g. "muc" appearing anywhere else in
// the tree could accidentally satisfy a rule meant only for muc/. No
// trailing "$" is added: the author decides whether the rule should cover
// one file or an entire subtree.
func ScopePathRegex(dir, pattern string) string {
	if dir == "." {
		return pattern
	}

	anchored := strings.HasPrefix(pattern, "^")
	pattern = strings.TrimPrefix(pattern, "^")
	pattern = strings.TrimPrefix(pattern, "/")

	prefix := "^" + regexp.QuoteMeta(dir) + "/"
	if anchored {
		return prefix + pattern
	}
	return prefix + ".*" + pattern
}
