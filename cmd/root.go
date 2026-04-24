package cmd

import (
	"os"

	"github.com/ValentinGerlach/oink/cmd/environments"
	"github.com/ValentinGerlach/oink/pkg/logging"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

var rootCmd = &cobra.Command{
	Use:   "oink",
	Short: "Run local OpenControlPlane environments in kind clusters",
	Long:  `oink is a CLI tool for running local OpenControlPlane environments in kind clusters, for local development or CI.`,
}

func Execute() {
	ctx := ctrl.SetupSignalHandler()

	logger := logging.NewLogger()
	ctx = logging.IntoContext(ctx, logger)

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(environments.NewEnvironmentsCmd())
}
