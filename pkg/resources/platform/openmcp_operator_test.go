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

	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != 1 {
		t.Errorf("replicas = %v, want 1", deployment.Spec.Replicas)
	}
	if deployment.Spec.Selector.MatchLabels["app"] != "openmcp-operator" {
		t.Errorf("selector label got %q, want %q", deployment.Spec.Selector.MatchLabels["app"], "openmcp-operator")
	}
	if deployment.Spec.Template.Labels["app"] != "openmcp-operator" {
		t.Errorf("pod template label got %q, want %q", deployment.Spec.Template.Labels["app"], "openmcp-operator")
	}
	if deployment.Spec.Template.Spec.ServiceAccountName != sa.Name {
		t.Errorf("serviceAccountName = %q, want %q", deployment.Spec.Template.Spec.ServiceAccountName, sa.Name)
	}

	if len(deployment.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("init containers count = %d, want 1", len(deployment.Spec.Template.Spec.InitContainers))
	}
	init := deployment.Spec.Template.Spec.InitContainers[0]
	if init.Image != operatorImage {
		t.Errorf("init container image = %q, want %q", init.Image, operatorImage)
	}
	if !slices.Contains(init.Args, "init") {
		t.Errorf("container args %v do not contain init command", init.Args)
	}
	if !slices.Contains(init.Args, "test-env") {
		t.Errorf("container args %v do not contain environment", init.Args)
	}
	if !slices.Contains(init.Args, "/etc/openmcp-operator/config") {
		t.Errorf("container args %v do not config path", init.Args)
	}
	if init.VolumeMounts[0].MountPath != "/etc/openmcp-operator" {
		t.Errorf("init container volume mount path = %q, want %q", init.VolumeMounts[0].MountPath, "/etc/openmcp-operator")
	}
	if init.VolumeMounts[0].Name != "config" {
		t.Errorf("init container volume mount name = %q, want %q", init.VolumeMounts[0].Name, "config")
	}
	testPodEnv(t, "init", init.Env)

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("containers count = %d, want 1", len(deployment.Spec.Template.Spec.Containers))
	}
	main := deployment.Spec.Template.Spec.Containers[0]
	if main.Image != operatorImage {
		t.Errorf("container image = %q, want %q", main.Image, operatorImage)
	}
	if !slices.Contains(main.Args, "run") {
		t.Errorf("container args %v do not contain init command", main.Args)
	}
	if !slices.Contains(main.Args, "test-env") {
		t.Errorf("container args %v do not contain environment", main.Args)
	}
	if !slices.Contains(main.Args, "/etc/openmcp-operator/config") {
		t.Errorf("container args %v do not config path", main.Args)
	}
	if main.VolumeMounts[0].MountPath != "/etc/openmcp-operator" {
		t.Errorf("init container volume mount path = %q, want %q", init.VolumeMounts[0].MountPath, "/etc/openmcp-operator")
	}
	if main.VolumeMounts[0].Name != "config" {
		t.Errorf("init container volume mount name = %q, want %q", init.VolumeMounts[0].Name, "config")
	}
	testPodEnv(t, "main", main.Env)

	if len(deployment.Spec.Template.Spec.Volumes) != 1 {
		t.Fatalf("volumes count = %d, want 1", len(deployment.Spec.Template.Spec.Volumes))
	}
	if deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name != cm.Name {
		t.Errorf("volume configmap = %q, want %q", deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Name, cm.Name)
	}
}

func testPodEnv(t *testing.T, containerName string, env []corev1.EnvVar) {
	t.Helper()
	want := []corev1.EnvVar{
		{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"}}},
		{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"}}},
		{Name: "POD_IP", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "status.podIP"}}},
		{Name: "POD_SERVICE_ACCOUNT_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.serviceAccountName"}}},
	}
	for _, w := range want {
		idx := slices.IndexFunc(env, func(e corev1.EnvVar) bool { return e.Name == w.Name })
		if idx == -1 {
			t.Errorf("%q container: missing env var %q", containerName, w.Name)
			continue
		}
		got := env[idx]
		if got.ValueFrom == nil || got.ValueFrom.FieldRef == nil {
			t.Errorf("%q container: env var %q has no FieldRef", containerName, w.Name)
			continue
		}
		if got.ValueFrom.FieldRef.APIVersion != w.ValueFrom.FieldRef.APIVersion {
			t.Errorf("%q container: env var %q APIVersion = %q, want %q", containerName, w.Name, got.ValueFrom.FieldRef.APIVersion, w.ValueFrom.FieldRef.APIVersion)
		}
		if got.ValueFrom.FieldRef.FieldPath != w.ValueFrom.FieldRef.FieldPath {
			t.Errorf("%q container: env var %q fieldPath = %q, want %q", containerName, w.Name, got.ValueFrom.FieldRef.FieldPath, w.ValueFrom.FieldRef.FieldPath)
		}
	}
}

func TestOperatorDeployment_ReadyFn(t *testing.T) {
	tests := []struct {
		name          string
		generation    int64
		observedGen   int64
		replicas      int32 // 0 means leave nil (ReadyFn defaults to 1)
		readyReplicas int32
		wantReady     bool
	}{
		{"ready with default replicas", 1, 1, 0, 1, true},
		{"ready with explicit replicas", 1, 1, 2, 2, true},
		{"not enough ready replicas", 1, 1, 0, 0, false},
		{"partially ready", 1, 1, 3, 2, false},
		{"generation mismatch", 2, 1, 0, 1, false},
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
			deployment.Status.ObservedGeneration = tt.observedGen
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
