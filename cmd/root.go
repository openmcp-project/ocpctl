package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

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
var timeout time.Duration
var cancelTimeout context.CancelFunc

func Execute() {
	ctx := ctrl.SetupSignalHandler()

	err := rootCmd.ExecuteContext(ctx)
	if cancelTimeout != nil {
		cancelTimeout()
	}
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 10*time.Minute, "maximum time to wait for the command to complete (0 means no timeout)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Configure timeout
		ctx := cmd.Context()
		if timeout > 0 {
			ctx, cancelTimeout = context.WithTimeout(ctx, timeout)
		}

		// Configure logger
		logger, err := logging.NewLogger(verbose)
		if err != nil {
			return fmt.Errorf("creating logger: %w", err)
		}
		ctx = logging.IntoContext(ctx, logger)

		// Set new context
		cmd.SetContext(ctx)
		return nil
	}
	rootCmd.AddCommand(environments.NewEnvironmentsCmd())
	rootCmd.AddCommand(cmdversion.NewVersionCmd())
	rootCmd.Version = version.Version()
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n", version.Version()))
}
