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
		{"subdir join and anchor", "muc", "secrets/.*", "^muc/secrets/.*"},
		{"nested subdir", "muc/prod", "db/.*", "^muc/prod/db/.*"},
		{"dir with regex metachars", "my.app", "secrets/.*", "^my\\.app/secrets/.*"},
		{"leading slash trimmed", "muc", "/secrets/.*", "^muc/secrets/.*"},
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
