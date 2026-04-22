package environments

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update an existing environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Updating environment %q (config: %s)\n", name, configFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "path to the environment config file")
	cmd.MarkFlagRequired("config")

	return cmd
}
