package integrate

import "github.com/spf13/cobra"

func NewIntegrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrate",
		Short: "Integrate an OpenControlPlane environment with external platforms",
	}

	cmd.AddCommand(newPlatformMeshCmd())

	return cmd
}
