package clusters

import (
	"context"
	"fmt"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
)

func Export(ctx context.Context, environment, clusterName, explicitPath string, internal bool, cp ClusterProvider) error {
	log := logging.FromContext(ctx)
	log.Debugf("retrieving clusters of environment: %s", environment)
	clusters, managed, err := getClusters(ctx, environment, cp)
	if err != nil {
		return fmt.Errorf("could not fetch clusters of environment %s: %w", environment, err)
	}
	if !managed {
		return fmt.Errorf("environment %s is not managed by ocpctl", environment)
	}
	log.Debugf("exporting kubeconfig for cluster %s/%s", environment, clusterName)
	err = exportKubeconfig(clusters, clusterName, explicitPath, internal, cp)
	if err != nil {
		return fmt.Errorf("could not export kubeconfig for cluster %s/%s: %w", environment, clusterName, err)
	}
	return nil
}

func exportKubeconfig(clusters []clustersv1alpha1.Cluster, clusterName string, explicitPath string, internal bool, cp ClusterProvider) error {
	for _, c := range clusters {
		if c.Name == clusterName {
			name, err := kind.KindClusterName(c)
			if err != nil {
				return err
			}
			err = cp.ExportKubeconfig(name, explicitPath, internal)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("cluster %s not found", clusterName)
}

func Get(ctx context.Context, environment, clusterName string, internal bool, cp ClusterProvider) (string, error) {
	log := logging.FromContext(ctx)
	log.Debugf("retrieving clusters of environment: %s", environment)
	clusters, managed, err := getClusters(ctx, environment, cp)
	if err != nil {
		return "", fmt.Errorf("could not fetch clusters of environment %s: %w", environment, err)
	}
	if !managed {
		return "", fmt.Errorf("environment %q is not managed by ocpctl", environment)
	}
	log.Debugf("retrieving kubeconfig of: %s/%s", environment, clusterName)
	config, err := getKubeconfig(clusters, clusterName, internal, cp)
	if err != nil {
		return "", fmt.Errorf("could not retrieve kubeconfig for %s/%s: %w", environment, clusterName, err)
	}
	return config, nil
}

func getKubeconfig(clusters []clustersv1alpha1.Cluster, clusterName string, internal bool, cp ClusterProvider) (string, error) {
	for _, c := range clusters {
		if c.Name == clusterName {
			name, err := kind.KindClusterName(c)
			if err != nil {
				return "", err
			}
			config, err := cp.GetKubeconfig(name, internal)
			if err != nil {
				return "", err
			}
			return config, nil
		}
	}
	return "", fmt.Errorf("cluster %s not found", clusterName)
}
