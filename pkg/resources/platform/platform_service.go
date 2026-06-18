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

// PlatformService returns a Resource for a PlatformService custom resource.
// Depends on the operator deployment being ready.
func PlatformService(name, image string, deployment *appsv1.Deployment) *resources.Resource {
	ps := &v1alpha1.PlatformService{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	return &resources.Resource{
		Object:       ps,
		Dependencies: []client.Object{deployment},
		ReadyFn: func(ctx context.Context) (bool, error) {
			if ps.Status.ObservedGeneration != ps.Generation {
				return false, nil
			}
			return ps.Status.Phase == commonapi.StatusPhaseReady, nil
		},
		MutateFn: func(_ context.Context) error {
			ps.Spec = v1alpha1.PlatformServiceSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Image: image,
				},
			}
			return nil
		},
	}
}
