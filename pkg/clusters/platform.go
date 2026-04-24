package clusters

import (
	"fmt"

	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	providerv1alpha1 "github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// CreatePlatformCluster creates the kind cluster for the platform components.
// The cluster name is scoped to the given environment so multiple OinK
// environments can coexist on the same machine.
func CreatePlatformCluster(environment string) error {
	name := platformClusterName(environment)
	provider := cluster.NewProvider()
	return provider.Create(name, cluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{
		Nodes: []v1alpha4.Node{
			{
				Role: v1alpha4.ControlPlaneRole,
				ExtraMounts: []v1alpha4.Mount{
					{
						HostPath:      "/var/run/docker.sock",
						ContainerPath: "/var/run/host-docker.sock",
					},
				},
			},
		},
	}))
}

// PlatformClusterClient returns a controller-runtime client for the platform cluster of the given environment.
func PlatformClusterClient(environment string) (client.Client, error) {
	name := platformClusterName(environment)
	provider := cluster.NewProvider()

	kubeconfig, err := provider.KubeConfig(name, false)
	if err != nil {
		return nil, fmt.Errorf("getting kubeconfig for cluster %q: %w", name, err)
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("parsing kubeconfig for cluster %q: %w", name, err)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := clustersv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := providerv1alpha1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return client.New(restConfig, client.Options{Scheme: scheme})
}

func platformClusterName(environment string) string {
	return fmt.Sprintf("%s-platform", environment)
}
