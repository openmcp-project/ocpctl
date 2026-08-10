package clusters

import "sigs.k8s.io/controller-runtime/pkg/client"

type ClusterProvider interface {
	PlatformClusterClient(name string) (client.Client, error)
	ListClusters() ([]string, error)
	GetKubeconfig(name string, internal bool) (string, error)
	ExportKubeconfig(name string, explicitPath string, internal bool) error
}
