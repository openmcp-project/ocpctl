package environments

import (
	"github.com/ValentinGerlach/oink/pkg/logging"
	"github.com/ValentinGerlach/oink/pkg/versions"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var operatorImage string
	var clusterProviderImage string

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an existing environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := logging.FromContext(ctx)
			name := args[0]

			if err := applyPlatformResources(ctx, name, operatorImage, clusterProviderImage); err != nil {
				return err
			}

			log.Infof("Environment %q updated successfully", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&operatorImage, "operator-image", versions.Operator().String(), "container image for the openmcp-operator")
	cmd.Flags().StringVar(&clusterProviderImage, "cluster-provider-image", versions.ClusterProviderKind().String(), "container image for the kind cluster provider")

	return cmd
}
