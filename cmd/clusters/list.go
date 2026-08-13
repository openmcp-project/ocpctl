package clusters

import (
	"fmt"
	"sort"

	clusterspkg "github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			clusters, err := clusterspkg.List(ctx, kind.NewProviderKind())
			if err != nil {
				return err
			}

			envs := make([]string, 0, len(clusters))
			for env := range clusters {
				envs = append(envs, env)
			}
			sort.Strings(envs)

			for _, env := range envs {
				fmt.Println(env)
				clusterList := clusters[env]
				sort.Slice(clusterList, func(i, j int) bool {
					ri, rj := clusterRank(clusterList[i]), clusterRank(clusterList[j])
					if ri != rj {
						return ri < rj
					}
					return clusterList[i].Name < clusterList[j].Name
				})
				for i, c := range clusterList {
					if i < len(clusterList)-1 {
						fmt.Printf("  ├── %s\n", c.Name)
					} else {
						fmt.Printf("  └── %s\n", c.Name)
					}
				}
				fmt.Println()
			}
			return nil
		},
	}
}

var purposeOrder = map[string]int{
	clustersv1alpha1.PURPOSE_PLATFORM:   0,
	clustersv1alpha1.PURPOSE_ONBOARDING: 1,
	clustersv1alpha1.PURPOSE_MCP:        2,
	clustersv1alpha1.PURPOSE_WORKLOAD:   3,
}

func clusterRank(c clustersv1alpha1.Cluster) int {
	best := len(purposeOrder)
	for _, p := range c.Spec.Purposes {
		if r, ok := purposeOrder[p]; ok && r < best {
			best = r
		}
	}
	return best
}
