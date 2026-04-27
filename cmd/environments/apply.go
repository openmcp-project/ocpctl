package environments

import (
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/config"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "apply <name>",
		Short: "Create or update an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			log := logging.FromContext(ctx)
			name := args[0]

			cfg := config.Default()
			if configPath != "" {
				userCfg, err := config.Load(configPath)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				cfg = config.Merge(cfg, userCfg)
			}

			log.Infof("Ensuring platform cluster for environment %q", name)
			created, err := clusters.EnsurePlatformCluster(name)
			if err != nil {
				return fmt.Errorf("ensuring platform cluster: %w", err)
			}
			if created {
				log.Info("Platform cluster created")
			} else {
				log.Info("Platform cluster already exists")
			}

			clusterProviderImage := ""
			for _, cp := range cfg.Spec.ClusterProviders {
				if cp.Name == "kind" {
					clusterProviderImage = cp.Image
					break
				}
			}
			if clusterProviderImage == "" {
				return fmt.Errorf("no cluster provider for kind configured")
			}

			if err := applyPlatformResources(ctx, name, cfg.Spec.Operator.Image, clusterProviderImage); err != nil {
				return err
			}

			log.Infof("Environment %q applied successfully", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to environment config file")

	return cmd
}
