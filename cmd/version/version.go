package version

import (
	"fmt"

	internalversion "github.com/openmcp-project/ocpctl/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version of ocpctl",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(internalversion.Version())
		},
	}
}
