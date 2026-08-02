package cli

import (
	"fmt"

	"github.com/Kreibich04/sops-config/internal/merge"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check sops-config.yaml files under --root without writing .sops.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, diags, err := merge.Run(rootDir)
			if err != nil {
				return err
			}

			printDiagnostics(cmd.ErrOrStderr(), diags)
			if merge.HasErrors(diags) {
				return fmt.Errorf("validate: found errors above")
			}

			if len(result.Rules) == 0 && !force {
				return fmt.Errorf("validate: no rules resolved; generate would refuse to write without --force")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "ok: %d rules would be generated\n", len(result.Rules))
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "accept zero resolved rules (mirrors generate --force)")
	return cmd
}
