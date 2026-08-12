package clusters

import (
	"context"
	"fmt"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/providers/kind"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
)

func List(ctx context.Context, cp ClusterProvider) (map[string][]clustersv1alpha1.Cluster, error) {
	log := logging.FromContext(ctx)
	log.Debug("retrieving kind clusters")
	kindClusters, err := cp.ListClusters()
	if err != nil {
		return nil, fmt.Errorf("listing kind clusters: %w", err)
	}
	environments := make(map[string][]clustersv1alpha1.Cluster)
	for _, c := range kindClusters {
		if !strings.HasSuffix(c, kind.PlatformClusterSuffix) {
			continue
		}
		env := strings.TrimSuffix(c, kind.PlatformClusterSuffix)
		log.Debugf("retrieving clusters of environment %q", env)
		envClusters, managed, err := getClusters(ctx, env, cp)
		if err != nil {
			log.Warnf("skipping cluster %q: %v", c, err)
			continue
		}
		if !managed {
			log.Debug("cluster %q not managed by ocpctl", c)
			continue
		}
		environments[env] = envClusters
	}

	return environments, nil
}

func getClusters(ctx context.Context, environment string, cp ClusterProvider) ([]clustersv1alpha1.Cluster, bool, error) {
	c, err := cp.PlatformClusterClient(environment)
	if err != nil {
		return nil, false, err
	}
	cl := &clustersv1alpha1.ClusterList{}
	err = c.List(ctx, cl)
	if err != nil {
		if apimeta.IsNoMatchError(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return cl.Items, true, nil
}
