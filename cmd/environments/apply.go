package environments

import (
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/config"
	envpkg "github.com/openmcp-project/ocpctl/pkg/environments"
	"github.com/spf13/cobra"
)

func newApplyCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "apply <name>",
		Short: "Create or update an environment",
		Args:  validatedNameArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			cfg := config.Default()
			if configPath != "" {
				userCfg, err := config.Load(configPath)
				if err != nil {
					return fmt.Errorf("loading config: %w", err)
				}
				cfg = config.Merge(cfg, userCfg)
			}

			return envpkg.Apply(ctx, name, cfg, &clusters.ProviderKind{})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to environment config file")

	return cmd
}
