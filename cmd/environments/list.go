package environments

import (
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	envpkg "github.com/openmcp-project/ocpctl/pkg/environments"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all environments",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			environments, err := envpkg.List(ctx, kind.NewProviderKind())
			if err != nil {
				return err
			}

			if len(environments) == 0 {
				fmt.Println("No environments found.")
				return nil
			}

			for _, env := range environments {
				fmt.Println(env)
			}

			return nil
		},
	}
}
