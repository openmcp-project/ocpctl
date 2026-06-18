package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	commonapi "github.com/openmcp-project/openmcp-operator/api/common"
	"github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceProvider returns a Resource for a ServiceProvider custom resource.
// Depends on the operator deployment being ready.
func ServiceProvider(name, image string, deployment *appsv1.Deployment) *resources.Resource {
	sp := &v1alpha1.ServiceProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	return &resources.Resource{
		Object:       sp,
		Dependencies: []client.Object{deployment},
		ReadyFn: func(ctx context.Context) (bool, error) {
			if sp.Status.ObservedGeneration != sp.Generation {
				return false, nil
			}
			return sp.Status.Phase == commonapi.StatusPhaseReady, nil
		},
		MutateFn: func(_ context.Context) error {
			sp.Spec = v1alpha1.ServiceProviderSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Image: image,
				},
			}
			return nil
		},
	}
}
