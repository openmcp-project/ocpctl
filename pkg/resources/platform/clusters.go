package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	"github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PlatformCluster returns a Resource for the platform Cluster custom resource
// in the operator namespace. This resource registers the platform kind cluster
// with the openmcp-operator. Depends on the operator deployment being ready.
func PlatformCluster(environment string, ns *corev1.Namespace, deployment *appsv1.Deployment) *resources.Resource {
	cluster := &v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "platform",
			Namespace: ns.Name,
		},
	}

	return &resources.Resource{
		Object:       cluster,
		Dependencies: []client.Object{deployment},
		MutateFn: func(_ context.Context) error {
			metav1.SetMetaDataAnnotation(&cluster.ObjectMeta, "kind.clusters.openmcp.cloud/name", environment+"-platform")
			cluster.Spec = v1alpha1.ClusterSpec{
				Kubernetes: v1alpha1.K8sConfiguration{},
				Profile:    "kind",
				Purposes:   []string{v1alpha1.PURPOSE_PLATFORM},
				Tenancy:    v1alpha1.TENANCY_SHARED,
			}
			return nil
		},
	}
}
