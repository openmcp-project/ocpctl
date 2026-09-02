package environments

import (
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/config"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an environment config file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return fmt.Errorf("--config is required")
			}
			userCfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if err := userCfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}
			fmt.Printf("Config %q is valid.\n", configPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to environment config file")

	return cmd
}
