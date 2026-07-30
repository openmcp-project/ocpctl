package platform

import (
	"context"
	"slices"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorImage = "ghcr.io/openmcp-project/images/openmcp-operator:v1.3.0"
)

func TestOperatorDeployment_Dependencies(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	crbResource := OperatorClusterRoleBinding(sa)
	crb := crbResource.Object.(*rbacv1.ClusterRoleBinding)
	cmResource := OperatorConfigMap("test-env", ns)
	cm := cmResource.Object.(*corev1.ConfigMap)
	r := OperatorDeployment(operatorImage, "test-env", ns, sa, crb, cm)

	if len(r.Dependencies) != 4 {
		t.Errorf("expected dependencies to be of length 4, got %d", len(r.Dependencies))
	}
	if !slices.Contains(r.Dependencies, client.Object(ns)) {
		t.Errorf("expected dependencies to contain namespace, got %v", r.Dependencies)
	}
	if !slices.Contains(r.Dependencies, client.Object(sa)) {
		t.Errorf("expected dependencies to contain service account, got %v", r.Dependencies)
	}
	if !slices.Contains(r.Dependencies, client.Object(crb)) {
		t.Errorf("expected dependencies to contain cluster role binding, got %v", r.Dependencies)
	}
	if !slices.Contains(r.Dependencies, client.Object(cm)) {
		t.Errorf("expected dependencies to contain config map, got %v", r.Dependencies)
	}
}

func TestOperatorDeployment_MutateFn(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	crbResource := OperatorClusterRoleBinding(sa)
	crb := crbResource.Object.(*rbacv1.ClusterRoleBinding)
	cmResource := OperatorConfigMap("test-env", ns)
	cm := cmResource.Object.(*corev1.ConfigMap)
	r := OperatorDeployment(operatorImage, "test-env", ns, sa, crb, cm)
	deployment := r.Object.(*appsv1.Deployment)

	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	if deployment.Spec.Template.Spec.ServiceAccountName != sa.Name {
		t.Errorf("serviceAccountName = %q, want %q", deployment.Spec.Template.Spec.ServiceAccountName, sa.Name)
	}
	if len(deployment.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("init containers count = %d, want 1", len(deployment.Spec.Template.Spec.InitContainers))
	}
	if deployment.Spec.Template.Spec.InitContainers[0].Image != operatorImage {
		t.Errorf("init container image = %q, want %q", deployment.Spec.Template.Spec.InitContainers[0].Image, operatorImage)
	}
	if !slices.Contains(deployment.Spec.Template.Spec.InitContainers[0].Args, "test-env") {
		t.Errorf("init container args %v do not contain environment name", deployment.Spec.Template.Spec.InitContainers[0].Args)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("containers count = %d, want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != operatorImage {
		t.Errorf("container image = %q, want %q", deployment.Spec.Template.Spec.Containers[0].Image, operatorImage)
	}
	if !slices.Contains(deployment.Spec.Template.Spec.Containers[0].Args, "test-env") {
		t.Errorf("container args %v do not contain environment name", deployment.Spec.Template.Spec.Containers[0].Args)
	}
	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("volumes count = %d, want 1", len(deployment.Spec.Template.Spec.Volumes))
	}
	if deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name != cm.Name {
		t.Errorf("volume configmap = %q, want %q", deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name, cm.Name)
	}
}

func TestOperatorDeployment_ReadyFn(t *testing.T) {
	tests := []struct {
		name               string
		generation         int64
		observedGeneration int64
		replicas           int32
		readyReplicas      int32
		wantReady          bool
	}{
		{
			name:               "ready with default replicas",
			generation:         1,
			observedGeneration: 1,
			readyReplicas:      1,
			wantReady:          true,
		},
		{
			name:               "ready with explicit replicas",
			generation:         1,
			observedGeneration: 1,
			replicas:           2,
			readyReplicas:      2,
			wantReady:          true,
		},
		{
			name:               "not enough ready replicas",
			generation:         1,
			observedGeneration: 1,
			wantReady:          false,
		},
		{
			name:               "partially ready",
			generation:         1,
			observedGeneration: 1,
			replicas:           3,
			readyReplicas:      2,
			wantReady:          false,
		},
		{
			name:               "generation mismatch",
			generation:         2,
			observedGeneration: 1,
			readyReplicas:      1,
			wantReady:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nsResource := OperatorNamespace("test-ns")
			ns := nsResource.Object.(*corev1.Namespace)
			saResource := OperatorServiceAccount(ns)
			sa := saResource.Object.(*corev1.ServiceAccount)
			crbResource := OperatorClusterRoleBinding(sa)
			crb := crbResource.Object.(*rbacv1.ClusterRoleBinding)
			cmResource := OperatorConfigMap("test-env", ns)
			cm := cmResource.Object.(*corev1.ConfigMap)

			r := OperatorDeployment(operatorImage, "test-env", ns, sa, crb, cm)
			deployment := r.Object.(*appsv1.Deployment)

			deployment.Generation = tt.generation
			deployment.Status.ObservedGeneration = tt.observedGeneration
			if tt.replicas > 0 {
				deployment.Spec.Replicas = &tt.replicas
			}
			deployment.Status.ReadyReplicas = tt.readyReplicas

			got, err := r.ReadyFn(context.Background())
			if err != nil {
				t.Fatalf("ReadyFn error: %v", err)
			}
			if got != tt.wantReady {
				t.Errorf("ready = %v, want %v", got, tt.wantReady)
			}
		})
	}
}
