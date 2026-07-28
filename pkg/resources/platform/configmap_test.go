package platform

import (
	"context"
	"strings"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	corev1 "k8s.io/api/core/v1"
)

func TestOperatorConfigMap_Dependencies(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	r := OperatorConfigMap("test-env", ns)

	if len(r.Dependencies) != 1 || r.Dependencies[0] != ns {
		t.Errorf("expected namespace as sole dependency, got %v", r.Dependencies)
	}
}

func TestOperatorConfigMap_MutateFn(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	r := OperatorConfigMap("test-env", ns)

	cm := r.Object.(*corev1.ConfigMap)
	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	if cm.Name != "openmcp-operator" {
		t.Errorf("name = %q, want %q", cm.Name, "openmcp-operator")
	}
	if cm.Namespace != ns.Name {
		t.Errorf("namespace = %q, want %q", cm.Namespace, ns.Name)
	}
	config, ok := cm.Data["config"]
	if !ok {
		t.Fatal("configmap missing 'config' key")
	}
	if !strings.Contains(config, "managedControlPlane") {
		t.Error("config missing 'managedControlPlane' section")
	}
	if !strings.Contains(config, "scheduler") {
		t.Error("config missing 'scheduler' section")
	}
	onboardingName := clusters.OnboardingClusterName("test-env")
	if !strings.Contains(config, onboardingName) {
		t.Errorf("config does not contain onboarding cluster name %q", onboardingName)
	}
}
