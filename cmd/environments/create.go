package environments

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Creating environment %q (config: %s)\n", name, configFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "path to the environment config file")

	return cmd
}
