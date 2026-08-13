package clusters

import (
	"context"
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/providers"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
)

func Export(ctx context.Context, environment, clusterName, explicitPath string, internal bool, cp providers.ClusterProvider) error {
	log := logging.FromContext(ctx)
	log.Debugf("retrieving clusters of environment %q", environment)
	clusters, managed, err := getClusters(ctx, environment, cp)
	if err != nil {
		return fmt.Errorf("could not fetch clusters of environment %q: %w", environment, err)
	}
	if !managed {
		return fmt.Errorf("environment %q is not managed by ocpctl", environment)
	}
	log.Debugf("exporting kubeconfig for cluster %s/%s", environment, clusterName)
	err = exportKubeconfig(clusters, clusterName, explicitPath, internal, cp)
	if err != nil {
		return fmt.Errorf("could not export kubeconfig for cluster %s/%s: %w", environment, clusterName, err)
	}
	return nil
}

func exportKubeconfig(clusters []clustersv1alpha1.Cluster, clusterName string, explicitPath string, internal bool, cp providers.ClusterProvider) error {
	name, err := resolveKindName(clusters, clusterName)
	if err != nil {
		return err
	}
	return cp.ExportKubeconfig(name, explicitPath, internal)
}

func Get(ctx context.Context, environment, clusterName string, internal bool, cp providers.ClusterProvider) (string, error) {
	log := logging.FromContext(ctx)
	log.Debugf("retrieving clusters of environment %q", environment)
	clusters, managed, err := getClusters(ctx, environment, cp)
	if err != nil {
		return "", fmt.Errorf("could not fetch clusters of environment %q: %w", environment, err)
	}
	if !managed {
		return "", fmt.Errorf("environment %q is not managed by ocpctl", environment)
	}
	log.Debugf("retrieving kubeconfig of %s/%s", environment, clusterName)
	config, err := getKubeconfig(clusters, clusterName, internal, cp)
	if err != nil {
		return "", fmt.Errorf("could not retrieve kubeconfig for %s/%s: %w", environment, clusterName, err)
	}
	return config, nil
}

func getKubeconfig(clusters []clustersv1alpha1.Cluster, clusterName string, internal bool, cp providers.ClusterProvider) (string, error) {
	name, err := resolveKindName(clusters, clusterName)
	if err != nil {
		return "", err
	}
	return cp.GetKubeconfig(name, internal)
}

func resolveKindName(clusters []clustersv1alpha1.Cluster, clusterName string) (string, error) {
	for _, c := range clusters {
		if c.Name == clusterName {
			return kind.KindClusterName(c)
		}
	}
	return "", fmt.Errorf("cluster not found")
}
