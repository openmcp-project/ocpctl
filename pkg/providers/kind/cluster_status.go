package kind

import (
	"fmt"

	"github.com/openmcp-project/cluster-provider-kind/api/v1alpha1"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
)

// KindClusterName extracts the kind cluster name from a Cluster's provider status.
func KindClusterName(cluster clustersv1alpha1.Cluster) (string, error) {
	if cluster.Status.ProviderStatus == nil {
		return "", fmt.Errorf("cluster %q has no provider status", cluster.Name)
	}
	var ps v1alpha1.ClusterStatus
	if err := cluster.Status.GetProviderStatus(&ps); err != nil {
		return "", fmt.Errorf("failed to parse provider status for cluster %q: %w", cluster.Name, err)
	}
	if ps.Kind != "ClusterStatus" || ps.APIVersion != v1alpha1.SchemeGroupVersion.String() {
		return "", fmt.Errorf("unexpected provider status type %s/%s for cluster %q", ps.APIVersion, ps.Kind, cluster.Name)
	}
	if ps.KindClusterName == "" {
		return "", fmt.Errorf("cluster %q has empty KindClusterName in provider status", cluster.Name)
	}
	return ps.KindClusterName, nil
}
