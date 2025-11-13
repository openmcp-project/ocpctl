package env

import (
	"fmt"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
)

func NewEnvironment(name string) *Environment {
	return &Environment{
		Name:         name,
		KindClusters: map[string]*KindCluster{},
	}
}

type Environment struct {
	Name         string
	KindClusters map[string]*KindCluster
}

func (e *Environment) GenerateKindClusterName(name string) string {
	return fmt.Sprintf("omink.%s.%s", e.Name, name)
}

type KindCluster struct {
	Config     *v1alpha4.Cluster
	KubeConfig string
	Client     client.Client
	RestConfig *rest.Config
}
