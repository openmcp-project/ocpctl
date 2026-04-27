package platform

import (
	"context"

	"github.com/ValentinGerlach/ocpctl/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorDeployment returns a Resource describing the openmcp-operator
// Deployment. The namespace, service account, ClusterRoleBinding, and ConfigMap
// must be applied before this one.
func OperatorDeployment(image, environment string, ns *corev1.Namespace, sa *corev1.ServiceAccount, crb *rbacv1.ClusterRoleBinding, cm *corev1.ConfigMap) *resources.Resource {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openmcp-operator",
			Namespace: ns.Name,
		},
	}

	return &resources.Resource{
		Object: deployment,
		Dependencies: []client.Object{
			ns,
			sa,
			crb,
			cm,
		},
		ReadyFn: func(ctx context.Context) (bool, error) {
			if deployment.Generation != deployment.Status.ObservedGeneration {
				return false, nil
			}
			desired := int32(1)
			if deployment.Spec.Replicas != nil {
				desired = *deployment.Spec.Replicas
			}
			return deployment.Status.ReadyReplicas >= desired, nil
		},
		MutateFn: func(_ context.Context) error {
			replicas := int32(1)
			podEnv := []corev1.EnvVar{
				{
					Name: "POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"},
					},
				},
				{
					Name: "POD_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"},
					},
				},
				{
					Name: "POD_IP",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"},
					},
				},
				{
					Name: "POD_SERVICE_ACCOUNT_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.serviceAccountName"},
					},
				},
			}
			configVolumeMount := corev1.VolumeMount{
				Name:      "config",
				MountPath: "/etc/openmcp-operator",
				ReadOnly:  true,
			}

			deployment.Spec = appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "openmcp-operator"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "openmcp-operator"},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: sa.Name,
						InitContainers: []corev1.Container{
							{
								Name:         "openmcp-operator-init",
								Image:        image,
								Args:         []string{"init", "--environment", environment, "--config", "/etc/openmcp-operator/config"},
								Env:          podEnv,
								VolumeMounts: []corev1.VolumeMount{configVolumeMount},
							},
						},
						Containers: []corev1.Container{
							{
								Name:         "openmcp-operator",
								Image:        image,
								Args:         []string{"run", "--environment", environment, "--config", "/etc/openmcp-operator/config"},
								Env:          podEnv,
								VolumeMounts: []corev1.VolumeMount{configVolumeMount},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
									},
								},
							},
						},
					},
				},
			}
			return nil
		},
	}
}
