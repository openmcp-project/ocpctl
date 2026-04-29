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
	rootCmd.AddCommand(cmdversion.NewVersionCmd())
	rootCmd.Version = version.Version()
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s\n", version.Version()))
}
