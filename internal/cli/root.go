// Package cli wires the sops-config subcommands to the merge pipeline.
package cli

import "github.com/spf13/cobra"

var rootDir string

// Version is the CLI version reported by --version. It is overridden at
// build time via -ldflags "-X main.version=...".
var Version = "dev"

// NewRootCmd builds the sops-config root cobra command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "sops-config",
		Short:         "Generate .sops.yaml from scattered sops-config.yaml files",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVarP(&rootDir, "root", "r", ".", "root directory to search for sops-config.yaml files")

	root.AddCommand(newGenerateCmd())
	root.AddCommand(newValidateCmd())

	return root
}

// Execute runs the CLI with os.Args.
func Execute() error {
	return NewRootCmd().Execute()
}
