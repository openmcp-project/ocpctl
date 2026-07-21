package provider

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GardenerConfig holds the configuration required by GardenerProvider.
type GardenerConfig struct {
	// Project is the Gardener project name.
	Project string
	// SeedName is the seed cluster to schedule the shoot on (optional).
	SeedName string
	// KubeconfigPath is the path to the Gardener API server kubeconfig.
	KubeconfigPath string
}

// GardenerProvider implements ClusterProvider for Gardener shoots.
// Configuration is passed via GardenerConfig.
type GardenerProvider struct {
	Config GardenerConfig
}

// Name returns the provider identifier "gardener".
func (g *GardenerProvider) Name() string { return "gardener" }

// EnsureCluster is not yet implemented.
func (g *GardenerProvider) EnsureCluster(ctx context.Context, name string) (Kubeconfig, bool, error) {
	return nil, false, fmt.Errorf("GardenerProvider.EnsureCluster: not yet implemented")
}

// GetCluster is not yet implemented.
func (g *GardenerProvider) GetCluster(ctx context.Context, name string) (ClusterInfo, error) {
	return ClusterInfo{}, fmt.Errorf("GardenerProvider.GetCluster: not yet implemented")
}

// DeleteCluster is not yet implemented.
func (g *GardenerProvider) DeleteCluster(ctx context.Context, name string) error {
	return fmt.Errorf("GardenerProvider.DeleteCluster: not yet implemented")
}

// NewClient is not yet implemented.
func (g *GardenerProvider) NewClient(ctx context.Context, name string, addToScheme ...func(*runtime.Scheme) error) (client.Client, error) {
	return nil, fmt.Errorf("GardenerProvider.NewClient: not yet implemented")
}
