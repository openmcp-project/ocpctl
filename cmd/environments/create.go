package environments

import (
	"fmt"

	"github.com/ValentinGerlach/oink/pkg/clusters"
	"github.com/ValentinGerlach/oink/pkg/logging"
	"github.com/ValentinGerlach/oink/pkg/versions"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var operatorImage string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := logging.FromContext(ctx)
			name := args[0]

			log.Infof("Creating platform cluster for environment %q", name)
			if err := clusters.CreatePlatformCluster(name); err != nil {
				return fmt.Errorf("creating platform cluster: %w", err)
			}

			if err := applyPlatformResources(ctx, name, operatorImage); err != nil {
				return err
			}

			log.Infof("Environment %q created successfully", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&operatorImage, "operator-image", versions.Operator().String(), "container image for the openmcp-operator")

	return cmd
}
