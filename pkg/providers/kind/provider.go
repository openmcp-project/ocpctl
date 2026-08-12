package kind

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/cluster"

	"github.com/openmcp-project/ocpctl/pkg/providers"
)

var _ providers.ClusterProvider = &providerKind{}

// providerKind implements providers.ClusterProvider
type providerKind struct {
	provider *cluster.Provider
}

func NewProviderKind() *providerKind {
	return &providerKind{provider: cluster.NewProvider()}
}

func (p *providerKind) EnsurePlatformCluster(name string) (bool, error) {
	return EnsurePlatformCluster(name, p.provider)
}

func (p *providerKind) PlatformClusterClient(name string) (client.Client, error) {
	return PlatformClusterClient(name, p.provider)
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
