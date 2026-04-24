package environments

import (
	"context"
	"fmt"
	"time"

	"github.com/ValentinGerlach/oink/pkg/clusters"
	"github.com/ValentinGerlach/oink/pkg/logging"
	"github.com/ValentinGerlach/oink/pkg/resources"
	"github.com/ValentinGerlach/oink/pkg/resources/platform"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// applyPlatformResources connects to the platform cluster for the given
// environment and applies all platform resources, retrying every 5 seconds
// until no resources are skipped due to unready dependencies.
func applyPlatformResources(ctx context.Context, environment, operatorImage string) error {
	log := logging.FromContext(ctx)

	log.Info("Building platform cluster client")
	c, err := clusters.PlatformClusterClient(environment)
	if err != nil {
		return fmt.Errorf("building platform cluster client: %w", err)
	}

	nsResource := platform.OperatorNamespace()
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := platform.OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	crbResource := platform.OperatorClusterRoleBinding(sa)
	crb := crbResource.Object.(*rbacv1.ClusterRoleBinding)
	cmResource := platform.OperatorConfigMap(ns)
	cm := cmResource.Object.(*corev1.ConfigMap)

	platformCluster := &resources.Cluster{Client: c}
	platformCluster.AddResources(
		nsResource,
		saResource,
		crbResource,
		cmResource,
		platform.OperatorDeployment(operatorImage, environment, ns, sa, crb, cm),
	)

	manager := &resources.Manager{}
	manager.AddClusters(platformCluster)

	log.Info("Applying platform resources")
	for {
		summary, err := manager.Apply(ctx)
		if err != nil {
			return fmt.Errorf("applying resources: %w", err)
		}
		log.Infof("Applied: %d, Skipped: %d", summary.Applied, summary.Skipped)
		if summary.Skipped == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}
