package environments

import (
	"context"
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/providers"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
)

// Delete deletes all kind clusters that belong to the given environment.
// It connects to the platform cluster, lists all Cluster resources, and collects
// the kind cluster names from provider statuses set by cluster-provider-kind.
func Delete(ctx context.Context, name string, cp providers.ClusterProvider) error {
	log := logging.FromContext(ctx)

	c, err := cp.PlatformClusterClient(name)
	if err != nil {
		return fmt.Errorf("connecting to platform cluster: %w", err)
	}

	clusterList := &clustersv1alpha1.ClusterList{}
	if err := c.List(ctx, clusterList); err != nil {
		return fmt.Errorf("listing clusters on platform cluster: %w", err)
	}

	var kindClusters []string
	for _, cl := range clusterList.Items {
		name, err := kind.KindClusterName(cl)
		if err != nil {
			log.Warnf("Skipping cluster %q: %v", cl.Name, err)
			continue
		}
		kindClusters = append(kindClusters, name)
	}

	platformCluster := kind.PlatformClusterName(name)
	sortPlatformLast(kindClusters, platformCluster)

	for _, kindCluster := range kindClusters {
		log.Infof("Deleting kind cluster %q", kindCluster)
		if err := cp.DeleteCluster(kindCluster); err != nil {
			return fmt.Errorf("deleting kind cluster %q: %w", kindCluster, err)
		}
	}

	return nil
}

// sortPlatformLast moves the platform cluster to the end of the slice so it is deleted last.
func sortPlatformLast(clusters []string, platformCluster string) {
	for i, c := range clusters {
		if c == platformCluster {
			clusters[i], clusters[len(clusters)-1] = clusters[len(clusters)-1], clusters[i]
			return
		}
	}
}
