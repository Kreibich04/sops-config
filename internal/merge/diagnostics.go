// Package merge combines the Discovered sops-config.yaml files produced by
// internal/config into a single, scoped set of SOPS creation rules.
package merge

import "fmt"

// Level indicates how serious a Diagnostic is.
type Level int

const (
	// Warning diagnostics are surfaced but do not fail generate/validate.
	Warning Level = iota
	// Error diagnostics fail validate, and cause generate to refuse to write.
	Error
)

func (l Level) String() string {
	if l == Error {
		return "error"
	}
	return "warning"
}

// Diagnostic is a single problem found while merging configs, attributed to
// the source file that caused it.
type Diagnostic struct {
	Level   Level
	Source  string
	Message string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s: %s: %s", d.Level, d.Source, d.Message)
}

// HasErrors reports whether any diagnostic in the slice is Error-level.
func HasErrors(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Level == Error {
			return true
		}
	}
	return false
}
