package platform

import (
	"context"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/clusters"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPlatformCluster_Dependencies(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := PlatformCluster("test-env", ns, deployment)

	if len(r.Dependencies) != 1 || r.Dependencies[0] != deployment {
		t.Errorf("expected deployment as sole dependency, got %v", r.Dependencies)
	}
}

func TestPlatformCluster_MutateFn(t *testing.T) {
	nsResource := OperatorNamespace("test-ns")
	ns := nsResource.Object.(*corev1.Namespace)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := PlatformCluster("test-env", ns, deployment)

	cluster := r.Object.(*clustersv1alpha1.Cluster)
	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	if cluster.Name != "platform" {
		t.Errorf("name = %q, want %q", cluster.Name, "platform")
	}
	if cluster.Namespace != ns.Name {
		t.Errorf("namespace = %q, want %q", cluster.Namespace, ns.Name)
	}
	if cluster.Spec.Profile != "kind" {
		t.Errorf("profile = %q, want %q", cluster.Spec.Profile, "kind")
	}
	if cluster.Spec.Tenancy != clustersv1alpha1.TENANCY_SHARED {
		t.Errorf("tenancy = %q, want %q", cluster.Spec.Tenancy, clustersv1alpha1.TENANCY_SHARED)
	}
	if len(cluster.Spec.Purposes) != 1 || cluster.Spec.Purposes[0] != clustersv1alpha1.PURPOSE_PLATFORM {
		t.Errorf("purposes = %v, want %v", cluster.Spec.Purposes, []string{clustersv1alpha1.PURPOSE_PLATFORM})
	}

	expectedAnnotation := clusters.PlatformClusterName("test-env")
	gotAnnotation := cluster.Annotations["kind.clusters.openmcp.cloud/name"]
	if gotAnnotation != expectedAnnotation {
		t.Errorf("kind cluster name annotation = %q, want %q", gotAnnotation, expectedAnnotation)
	}
}

func TestPlatformCluster_ReadyFn(t *testing.T) {
	tests := []struct {
		name        string
		generation  int64
		observedGen int64
		phase       string
		wantReady   bool
	}{
		{"ready", 1, 1, clustersv1alpha1.CLUSTER_PHASE_READY, true},
		{"not ready", 1, 1, clustersv1alpha1.CLUSTER_PHASE_NOT_READY, false},
		{"generation mismatch", 2, 1, clustersv1alpha1.CLUSTER_PHASE_READY, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nsResource := OperatorNamespace("test-ns")
			ns := nsResource.Object.(*corev1.Namespace)
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
			r := PlatformCluster("test-env", ns, deployment)

			cluster := r.Object.(*clustersv1alpha1.Cluster)
			cluster.Generation = tt.generation
			cluster.Status.ObservedGeneration = tt.observedGen
			cluster.Status.Phase = tt.phase

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
