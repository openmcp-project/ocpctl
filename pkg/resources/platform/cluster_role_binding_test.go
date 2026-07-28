package platform

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestOperatorClusterRoleBinding_Dependencies(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	r := OperatorClusterRoleBinding(sa)

	if len(r.Dependencies) != 1 || r.Dependencies[0] != sa {
		t.Errorf("expected service account as sole dependency, got %v", r.Dependencies)
	}
}

func TestOperatorClusterRoleBinding_MutateFn(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	saResource := OperatorServiceAccount(ns)
	sa := saResource.Object.(*corev1.ServiceAccount)
	r := OperatorClusterRoleBinding(sa)

	crb := r.Object.(*rbacv1.ClusterRoleBinding)
	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	if crb.Name != "openmcp-operator" {
		t.Errorf("name = %q, want %q", crb.Name, "openmcp-operator")
	}
	if crb.RoleRef.Kind != "ClusterRole" {
		t.Errorf("roleRef.kind = %q, want %q", crb.RoleRef.Kind, "ClusterRole")
	}
	if crb.RoleRef.Name != "cluster-admin" {
		t.Errorf("roleRef.name = %q, want %q", crb.RoleRef.Name, "cluster-admin")
	}
	if crb.RoleRef.APIGroup != rbacv1.GroupName {
		t.Errorf("roleRef.apiGroup = %q, want %q", crb.RoleRef.APIGroup, rbacv1.GroupName)
	}

	if len(crb.Subjects) != 1 {
		t.Errorf("subjects count = %d, want 1", len(crb.Subjects))
	}
	subject := crb.Subjects[0]
	if subject.Kind != rbacv1.ServiceAccountKind {
		t.Errorf("subject.kind = %q, want %q", subject.Kind, rbacv1.ServiceAccountKind)
	}
	if subject.Name != sa.Name {
		t.Errorf("subject.name = %q, want %q", subject.Name, sa.Name)
	}
	if subject.Namespace != sa.Namespace {
		t.Errorf("subject.namespace = %q, want %q", subject.Namespace, sa.Namespace)
	}
}
