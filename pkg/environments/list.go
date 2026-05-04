package environments

import (
	"context"
	"fmt"
	"strings"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/kind/pkg/cluster"
)

// List returns the names of all ocpctl-managed local environments.
// It lists kind clusters with the "-platform" suffix and verifies each has the Cluster CRD installed.
func List(ctx context.Context) ([]string, error) {
	log := logging.FromContext(ctx)

	provider := cluster.NewProvider()
	kindClusters, err := provider.List()
	if err != nil {
		return nil, fmt.Errorf("listing kind clusters: %w", err)
	}

	var environments []string
	for _, c := range kindClusters {
		if !strings.HasSuffix(c, "-platform") {
			continue
		}
		env := strings.TrimSuffix(c, "-platform")

		managed, err := isOcpctlManaged(ctx, env)
		if err != nil {
			log.Warnf("Skipping cluster %q: %v", c, err)
			continue
		}
		if managed {
			environments = append(environments, env)
		}
	}

	return environments, nil
}

func isOcpctlManaged(ctx context.Context, environment string) (bool, error) {
	c, err := clusters.PlatformClusterClient(environment)
	if err != nil {
		return false, err
	}

	err = c.List(ctx, &clustersv1alpha1.ClusterList{})
	if err != nil {
		if apimeta.IsNoMatchError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
