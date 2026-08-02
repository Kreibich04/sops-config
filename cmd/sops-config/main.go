// Command sops-config merges sops-config.yaml files scattered across a
// directory tree into a single .sops.yaml.
package main

import (
	"fmt"
	"os"

	"github.com/Kreibich04/sops-config/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cli.Version = version
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
