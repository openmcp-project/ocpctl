package kind

import (
	"context"
	"fmt"
	"time"

	"github.com/ValentinGerlach/oink/pkg/env"
	"github.com/ValentinGerlach/oink/pkg/steps"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

func CreateCluster(config *v1alpha4.Cluster) []steps.Step {
	prov := cluster.NewProvider(
		cluster.ProviderWithDocker(),
	)

	return []steps.Step{
		{
			Description: fmt.Sprintf("Create kind cluster %s", config.Name),
			Run: func(ctx context.Context) error {
				options := []cluster.CreateOption{
					cluster.CreateWithV1Alpha4Config(config),
					cluster.CreateWithWaitForReady(1 * time.Minute),
				}
				return prov.Create(config.Name, options...)
			},
		},
		{
			Description: fmt.Sprintf("Export kubeconfig for kind cluster %s", config.Name),
			Run: func(ctx context.Context) error {
				kubeconfig, err := prov.KubeConfig(config.Name, false)
				if err != nil {
					return err
				}

				restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
				if err != nil {
					return err
				}

				c, err := client.New(restConfig, client.Options{
					Scheme: scheme,
				})
				if err != nil {
					return err
				}

				e := env.FromContext(ctx)
				e.KindClusters[config.Name] = &env.KindCluster{
					Config:     config,
					KubeConfig: kubeconfig,
					RestConfig: restConfig,
					Client:     c,
				}
				return nil
			},
		},
	}
}
