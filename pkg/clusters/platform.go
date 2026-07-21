package clusters

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/provider"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	providerv1alpha1 "github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kindProvider is the cluster provider backing the platform cluster.
var kindProvider provider.ClusterProvider = &provider.KindProvider{}

// EnsurePlatformCluster creates the kind cluster for the platform components if
// it does not already exist. Returns true if the cluster was newly created.
func EnsurePlatformCluster(environment string) (bool, error) {
	name := PlatformClusterName(environment)
	_, created, err := kindProvider.EnsureCluster(context.Background(), name)
	if err != nil {
		return false, err
	}
	return created, nil
}

// PlatformClusterClient returns a controller-runtime client for the platform cluster of the given environment.
func PlatformClusterClient(environment string) (client.Client, error) {
	name := PlatformClusterName(environment)
	return kindProvider.NewClient(context.Background(), name,
		clientgoscheme.AddToScheme,
		clustersv1alpha1.AddToScheme,
		providerv1alpha1.AddToScheme,
	)
}

const PlatformClusterSuffix = "-platform"

func PlatformClusterName(environment string) string {
	return environment + PlatformClusterSuffix
}
