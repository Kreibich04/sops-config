package cli

import (
	"fmt"
	"io"

	"github.com/Kreibich04/sops-config/internal/merge"
)

func printDiagnostics(w io.Writer, diags []merge.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintln(w, d.String())
	}
}
