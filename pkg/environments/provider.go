package environments

import "sigs.k8s.io/controller-runtime/pkg/client"

type ClusterProvider interface {
	EnsurePlatformCluster(name string) (bool, error)
	PlatformClusterClient(name string) (client.Client, error)
	ListClusters() ([]string, error)
	DeleteCluster(name string) error
}
