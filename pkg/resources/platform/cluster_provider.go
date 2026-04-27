package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	"github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ClusterProvider(image string, deployment *appsv1.Deployment) *resources.Resource {
	provider := &v1alpha1.ClusterProvider{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kind",
		},
	}

	return &resources.Resource{
		Object:       provider,
		Dependencies: []client.Object{deployment},
		MutateFn: func(_ context.Context) error {
			provider.Spec = v1alpha1.ClusterProviderSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Image: image,
					ExtraVolumes: []corev1.Volume{
						{
							Name: "docker-socket",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/host-docker.sock",
									Type: ptr.To(corev1.HostPathSocket),
								},
							},
						},
					},
					ExtraVolumeMounts: []corev1.VolumeMount{
						{
							Name:      "docker-socket",
							MountPath: "/var/run/docker.sock",
						},
					},
				},
			}
			return nil
		},
	}
}
