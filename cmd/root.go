package cmd

import (
	"fmt"
	"os"

	"github.com/openmcp-project/ocpctl/cmd/environments"
	cmdversion "github.com/openmcp-project/ocpctl/cmd/version"
	"github.com/openmcp-project/ocpctl/internal/version"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

var rootCmd = &cobra.Command{
	Use:   "ocpctl",
	Short: "Run local OpenControlPlane environments in kind clusters",
	Long:  `ocpctl is a CLI tool for running local OpenControlPlane environments in kind clusters, for local development or CI.`,
}

var verbose bool

func Execute() {
	ctx := ctrl.SetupSignalHandler()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger, err := logging.NewLogger(verbose)
		if err != nil {
			return fmt.Errorf("creating logger: %w", err)
		}
		cmd.SetContext(logging.IntoContext(cmd.Context(), logger))
		return nil
	}
	rootCmd.AddCommand(environments.NewEnvironmentsCmd())
	rootCmd.AddCommand(cmdversion.NewVersionCmd())
	rootCmd.Version = version.Version()
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n", version.Version()))
}
