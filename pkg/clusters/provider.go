package clusters

import "sigs.k8s.io/controller-runtime/pkg/client"

type ClusterProvider interface {
	PlatformClusterClient(name string) (client.Client, error)
	ListClusters() ([]string, error)
}
