package kind

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/cluster"
)

// providerKind implements the environments.ClusterProvider interface
type providerKind struct {
	provider *cluster.Provider
}

func NewProviderKind() *providerKind {
	return &providerKind{provider: cluster.NewProvider()}
}

func (p *providerKind) EnsurePlatformCluster(name string) (bool, error) {
	return EnsurePlatformCluster(name)
}

func (p *providerKind) PlatformClusterClient(name string) (client.Client, error) {
	return PlatformClusterClient(name)
}

func (p *providerKind) ListClusters() ([]string, error) {
	return p.provider.List()
}

func (p *providerKind) DeleteCluster(name string) error {
	return p.provider.Delete(name, "")
}

func (p *providerKind) GetKubeconfig(name string, internal bool) (string, error) {
	return p.provider.KubeConfig(name, internal)
}

func (p *providerKind) ExportKubeconfig(name string, explicitPath string, internal bool) error {
	return p.provider.ExportKubeConfig(name, explicitPath, internal)
}
