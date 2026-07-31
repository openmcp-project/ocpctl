package platform

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestOperatorNamespace_Object(t *testing.T) {
	r := OperatorNamespace("my-namespace")
	ns := r.Object.(*corev1.Namespace)
	if ns.Name != "my-namespace" {
		t.Errorf("name = %q, want %q", ns.Name, "my-namespace")
	}
}

func TestOperatorNamespace_Dependencies(t *testing.T) {
	r := OperatorNamespace("my-namespace")
	if len(r.Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %v", r.Dependencies)
	}
}

func TestOperatorNamespace_ReadyFn(t *testing.T) {
	tests := []struct {
		name      string
		phase     corev1.NamespacePhase
		wantReady bool
	}{
		{"active", corev1.NamespaceActive, true},
		{"terminating", corev1.NamespaceTerminating, false},
		{"something-else", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := OperatorNamespace("my-namespace")
			ns := r.Object.(*corev1.Namespace)
			ns.Status.Phase = tt.phase

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
