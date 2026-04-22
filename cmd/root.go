package cmd

import (
	"os"

	"github.com/ValentinGerlach/oink/cmd/environments"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oink",
	Short: "Run local OpenControlPlane environments in kind clusters",
	Long:  `oink is a CLI tool for running local OpenControlPlane environments in kind clusters, for local development or CI.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(environments.NewEnvironmentsCmd())
}
