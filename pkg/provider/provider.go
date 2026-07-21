package provider

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Kubeconfig is raw kubeconfig YAML bytes.
type Kubeconfig []byte

// ClusterInfo holds basic info about an existing cluster.
type ClusterInfo struct {
	Name string
}

// ClusterProvider abstracts cluster lifecycle (create, get, delete).
// Implementations: KindProvider, GardenerProvider.
type ClusterProvider interface {
	// Name returns the provider identifier used in config (e.g. "kind", "gardener").
	Name() string
	// EnsureCluster creates the cluster if it does not exist.
	// Returns the kubeconfig, whether the cluster was newly created, and any error.
	EnsureCluster(ctx context.Context, name string) (Kubeconfig, bool, error)
	// GetCluster returns info about an existing cluster, or an error if it doesn't exist.
	GetCluster(ctx context.Context, name string) (ClusterInfo, error)
	// DeleteCluster deletes the named cluster.
	DeleteCluster(ctx context.Context, name string) error
	// NewClient builds a controller-runtime client for the named cluster.
	NewClient(ctx context.Context, name string, addToScheme ...func(*runtime.Scheme) error) (client.Client, error)
}
