package environments

import (
	"context"
	"fmt"
	"strings"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/providers"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

// List returns the names of all ocpctl-managed local environments.
// It lists kind clusters with the "-platform" suffix and verifies each has the Cluster CRD installed.
func List(ctx context.Context, cp providers.ClusterProvider) ([]string, error) {
	log := logging.FromContext(ctx)

	kindClusters, err := cp.ListClusters()
	if err != nil {
		return nil, fmt.Errorf("listing kind clusters: %w", err)
	}

	var environments []string
	for _, c := range kindClusters {
		if !strings.HasSuffix(c, kind.PlatformClusterSuffix) {
			continue
		}
		env := strings.TrimSuffix(c, kind.PlatformClusterSuffix)

		managed, err := isOcpctlManaged(ctx, env, cp)
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

func isOcpctlManaged(ctx context.Context, environment string, cp providers.ClusterProvider) (bool, error) {
	c, err := cp.PlatformClusterClient(environment)
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
