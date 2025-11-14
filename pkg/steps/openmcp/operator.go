package openmcp

import (
	"context"

	"github.com/ValentinGerlach/oink/pkg/env"
	"github.com/ValentinGerlach/oink/pkg/steps"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	_ "embed"
)

const (
	systemNamespace         = "openmcp-system"
	operatorName            = "openmcp-operator"
	operatorConfigMountDir  = "/etc/openmcp-operator"
	operatorConfigMountPath = operatorConfigMountDir + "/config"
)

var (
	//go:embed operator-config.yaml
	operatorConfig string

	operatorLabels = map[string]string{
		"app": operatorName,
	}
)

func DeployOperator(image string, platformCluster string) []steps.Step {
	clientFromCluster := func(ctx context.Context, cluster string) client.Client {
		e := env.FromContext(ctx)
		return e.KindClusters[cluster].Client
	}

	return []steps.Step{
		{
			Description: "Create system namespace",
			Run: func(ctx context.Context) error {
				c := clientFromCluster(ctx, platformCluster)
				return c.Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: systemNamespace,
					},
				})
			},
		},
		{
			Description: "Create ServiceAccount for openmcp-operator",
			Run: func(ctx context.Context) error {
				c := clientFromCluster(ctx, platformCluster)
				return c.Create(ctx, &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operatorName,
						Namespace: systemNamespace,
					},
				})
			},
		},
		{
			Description: "Create ClusterRoleBinding for openmcp-operator",
			Run: func(ctx context.Context) error {
				c := clientFromCluster(ctx, platformCluster)
				return c.Create(ctx, &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: operatorName,
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: rbacv1.GroupName,
						Kind:     "ClusterRole",
						Name:     "cluster-admin",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      rbacv1.ServiceAccountKind,
							Name:      operatorName,
							Namespace: systemNamespace,
						},
					},
				})
			},
		},
		{
			Description: "Create ConfigMap for openmcp-operator",
			Run: func(ctx context.Context) error {
				c := clientFromCluster(ctx, platformCluster)
				return c.Create(ctx, &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operatorName,
						Namespace: systemNamespace,
					},
					Data: map[string]string{
						"config": operatorConfig,
					},
				})
			},
		},
		{
			Description: "Create Deployment for openmcp-operator",
			Run: func(ctx context.Context) error {
				c := clientFromCluster(ctx, platformCluster)
				e := env.FromContext(ctx)
				return c.Create(ctx, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      operatorName,
						Namespace: systemNamespace,
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: ptr.To[int32](1),
						Selector: &metav1.LabelSelector{
							MatchLabels: operatorLabels,
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: operatorLabels,
							},
							Spec: corev1.PodSpec{
								ServiceAccountName: operatorName,
								InitContainers: []corev1.Container{
									{
										Image: image,
										Name:  operatorName + "-init",
										Args: []string{
											"init",
											"--environment=" + e.Name,
											"--config=" + operatorConfigMountPath,
										},
										Env: getOperatorEnv(),
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "config",
												MountPath: operatorConfigMountDir,
												ReadOnly:  true,
											},
										},
									},
								},
								Containers: []corev1.Container{
									{
										Image: image,
										Name:  operatorName,
										Args: []string{
											"run",
											"--environment=" + e.Name,
											"--config=" + operatorConfigMountPath,
										},
										Env: getOperatorEnv(),
										VolumeMounts: []corev1.VolumeMount{
											{
												Name:      "config",
												MountPath: operatorConfigMountDir,
												ReadOnly:  true,
											},
										},
									},
								},
								Volumes: []corev1.Volume{
									{
										Name: "config",
										VolumeSource: corev1.VolumeSource{
											ConfigMap: &corev1.ConfigMapVolumeSource{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: operatorName,
												},
											},
										},
									},
								},
							},
						},
					},
				})
			},
		},
	}
}

func getOperatorEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "status.podIP",
				},
			},
		},
		{
			Name: "POD_SERVICE_ACCOUNT_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.serviceAccountName",
				},
			},
		},
	}
}
