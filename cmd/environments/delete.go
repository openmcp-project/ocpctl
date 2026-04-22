package environments

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Deleting environment %q\n", name)
			return nil
		},
	}
}
