package cli

import (
	"fmt"
	"path/filepath"

	"github.com/Kreibich04/sops-config/internal/merge"
	"github.com/Kreibich04/sops-config/internal/sopsyaml"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var output string
	var force bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Merge sops-config.yaml files under --root into .sops.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, diags, err := merge.Run(rootDir)
			if err != nil {
				return err
			}

			printDiagnostics(cmd.ErrOrStderr(), diags)
			if merge.HasErrors(diags) {
				return fmt.Errorf("generate: aborting due to errors above, .sops.yaml not written")
			}

			if len(result.Rules) == 0 && !force {
				return fmt.Errorf("generate: no rules resolved; use --force to write an empty .sops.yaml anyway")
			}

			out := output
			if out == "" {
				out = filepath.Join(rootDir, ".sops.yaml")
			}
			if err := sopsyaml.WriteFile(out, result.Rules); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d rules)\n", out, len(result.Rules))
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "output path (default: <root>/.sops.yaml)")
	cmd.Flags().BoolVar(&force, "force", false, "write .sops.yaml even if no rules resolved")

	return cmd
}
