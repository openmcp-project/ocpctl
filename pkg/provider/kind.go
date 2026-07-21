package provider

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

// KindProvider implements ClusterProvider using local kind clusters.
type KindProvider struct{}

func init() {
	Register(&KindProvider{})
}

// Name returns the provider identifier "kind".
func (k *KindProvider) Name() string { return "kind" }

// EnsureCluster creates the kind cluster if it does not already exist.
// It returns the cluster kubeconfig, whether the cluster was newly created, and any error.
func (k *KindProvider) EnsureCluster(_ context.Context, name string) (Kubeconfig, bool, error) {
	provider := cluster.NewProvider()

	existing, err := provider.List()
	if err != nil {
		return nil, false, fmt.Errorf("listing kind clusters: %w", err)
	}

	created := true
	for _, c := range existing {
		if c == name {
			created = false
			break
		}
	}

	if created {
		if err := provider.Create(name, cluster.CreateWithV1Alpha4Config(&v1alpha4.Cluster{
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
		})); err != nil {
			return nil, false, err
		}
	}

	kubeconfig, err := provider.KubeConfig(name, false)
	if err != nil {
		return nil, created, fmt.Errorf("getting kubeconfig for cluster %q: %w", name, err)
	}

	return Kubeconfig(kubeconfig), created, nil
}

// GetCluster returns info about an existing kind cluster, or an error if it does not exist.
func (k *KindProvider) GetCluster(_ context.Context, name string) (ClusterInfo, error) {
	provider := cluster.NewProvider()

	existing, err := provider.List()
	if err != nil {
		return ClusterInfo{}, fmt.Errorf("listing kind clusters: %w", err)
	}
	for _, c := range existing {
		if c == name {
			return ClusterInfo{Name: name}, nil
		}
	}

	return ClusterInfo{}, fmt.Errorf("kind cluster %q not found", name)
}

// DeleteCluster deletes the named kind cluster.
func (k *KindProvider) DeleteCluster(_ context.Context, name string) error {
	provider := cluster.NewProvider()
	return provider.Delete(name, "")
}

// NewClient builds a controller-runtime client for the named kind cluster.
func (k *KindProvider) NewClient(_ context.Context, name string, addToScheme ...func(*runtime.Scheme) error) (client.Client, error) {
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
	for _, add := range addToScheme {
		if err := add(scheme); err != nil {
			return nil, err
		}
	}

	return client.New(restConfig, client.Options{Scheme: scheme})
}
