package environments

import "github.com/spf13/cobra"

func NewEnvironmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environments",
		Aliases: []string{"env"},
		Short:   "Manage OpenControlPlane environments",
	}

	cmd.AddCommand(
		newApplyCmd(),
		newListCmd(),
		newDeleteCmd(),
		newValidateCmd(),
	)

	return cmd
}
