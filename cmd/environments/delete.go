package environments

import (
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	envpkg "github.com/openmcp-project/ocpctl/pkg/environments"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an environment",
		Args:  validatedNameArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			return envpkg.Delete(cmd.Context(), args[0], &kind.ProviderKind{})
		},
	}
}
