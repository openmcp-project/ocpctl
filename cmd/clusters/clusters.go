package clusters

import "github.com/spf13/cobra"

func NewClustersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clusters",
		Short: "Manage OpenControlPlane clusters",
	}

	cmd.AddCommand(
		newListCmd(),
	)

	return cmd
}
