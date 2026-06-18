package environments

import (
	"context"
	"fmt"
	"time"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	"github.com/openmcp-project/ocpctl/pkg/config"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/resources"
	"github.com/openmcp-project/ocpctl/pkg/resources/platform"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func Apply(ctx context.Context, name string, cfg *config.Environment) error {
	log := logging.FromContext(ctx)

	log.Infof("Ensuring platform cluster for environment %q", name)
	created, err := clusters.EnsurePlatformCluster(name)
	if err != nil {
		return fmt.Errorf("ensuring platform cluster: %w", err)
	}
	if created {
		log.Info("Platform cluster created")
	} else {
		log.Info("Platform cluster already exists")
	}

	if err := applyPlatformResources(ctx, name, cfg); err != nil {
		return err
	}

	log.Infof("Environment %q applied successfully", name)
	return nil
}

func applyPlatformResources(ctx context.Context, environment string, cfg *config.Environment) error {
	log := logging.FromContext(ctx)

	clusterProviderImage := ""
	for _, cp := range cfg.Spec.ClusterProviders {
		if cp.Name == "kind" {
			clusterProviderImage = cp.Image
			break
		}
	}
	if clusterProviderImage == "" {
		return fmt.Errorf("no cluster provider for kind configured")
	}
	if cfg.Spec.Namespace == "" {
		return fmt.Errorf("namespace must not be empty")
	}

	log.Info("Building platform cluster client")
	c, err := clusters.PlatformClusterClient(environment)
	if err != nil {
		return fmt.Errorf("building platform cluster client: %w", err)
	}

	nsResource := platform.OperatorNamespace(cfg.Spec.Namespace)
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := platform.OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	crbResource := platform.OperatorClusterRoleBinding(sa)
	crb := crbResource.Object.(*rbacv1.ClusterRoleBinding)
	cmResource := platform.OperatorConfigMap(environment, ns)
	cm := cmResource.Object.(*corev1.ConfigMap)
	deploymentResource := platform.OperatorDeployment(cfg.Spec.Operator.Image, environment, ns, sa, crb, cm)
	deployment := deploymentResource.Object.(*appsv1.Deployment)

	platformCluster := &resources.Cluster{Client: c}
	platformCluster.AddResources(
		nsResource,
		saResource,
		crbResource,
		cmResource,
		deploymentResource,
		platform.PlatformCluster(environment, ns, deployment),
		platform.ClusterProvider(clusterProviderImage, deployment),
	)
	for _, sp := range cfg.Spec.ServiceProviders {
		platformCluster.AddResources(platform.ServiceProvider(sp.Name, sp.Image, deployment))
	}
	for _, ps := range cfg.Spec.PlatformServices {
		platformCluster.AddResources(platform.PlatformService(ps.Name, ps.Image, deployment))
	}

	manager := &resources.Manager{}
	manager.AddClusters(platformCluster)

	log.Info("Applying platform resources")
	for {
		summary, err := manager.Apply(ctx)
		if err != nil {
			return fmt.Errorf("applying resources: %w", err)
		}
		log.Infof("Applied: %d/%d, Ready: %d/%d, Waiting for dependencies: %d", len(summary.Applied), summary.Total(), len(summary.Ready), summary.Total(), len(summary.WaitingForDeps))
		if len(summary.Ready) == summary.Total() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}
