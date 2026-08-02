package merge

import "testing"

func TestScopePathRegex(t *testing.T) {
	cases := []struct {
		name    string
		dir     string
		pattern string
		want    string
	}{
		{"root passthrough", ".", ".*/muc/secrets/.*", ".*/muc/secrets/.*"},
		{"subdir join, unanchored search within dir", "muc", "secrets/.*", "^muc/.*secrets/.*"},
		{"subdir join, explicit anchor", "muc", "^secrets/.*", "^muc/secrets/.*"},
		{"nested subdir, unanchored", "muc/prod", "db/.*", "^muc/prod/.*db/.*"},
		{"nested subdir, anchored", "muc/prod", "^db/.*", "^muc/prod/db/.*"},
		{"dir with regex metachars", "my.app", "secrets/.*", "^my\\.app/.*secrets/.*"},
		{"leading slash trimmed, unanchored", "muc", "/secrets/.*", "^muc/.*secrets/.*"},
		{"leading slash trimmed, anchored", "muc", "^/secrets/.*", "^muc/secrets/.*"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopePathRegex(tc.dir, tc.pattern)
			if got != tc.want {
				t.Errorf("ScopePathRegex(%q, %q) = %q, want %q", tc.dir, tc.pattern, got, tc.want)
			}
		})
	}
}
