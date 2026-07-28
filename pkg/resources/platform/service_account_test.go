package platform

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestOperatorServiceAccount_Object(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	r := OperatorServiceAccount(ns)

	sa := r.Object.(*corev1.ServiceAccount)
	if sa.Name != "openmcp-operator" {
		t.Errorf("name = %q, want %q", sa.Name, "openmcp-operator")
	}
	if sa.Namespace != ns.Name {
		t.Errorf("namespace = %q, want %q", sa.Namespace, ns.Name)
	}
}

func TestOperatorServiceAccount_Dependencies(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	r := OperatorServiceAccount(ns)

	if len(r.Dependencies) != 1 || r.Dependencies[0] != ns {
		t.Errorf("expected namespace as sole dependency, got %v", r.Dependencies)
	}
}
