package platform

import (
	"context"
	"testing"

	commonapi "github.com/openmcp-project/openmcp-operator/api/common"
	"github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPlatformService_MutateFn(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := PlatformService("gateway", "ghcr.io/openmcp-project/images/platform-service-gateway:v0.0.12", deployment)

	ps := r.Object.(*v1alpha1.PlatformService)
	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}
	if ps.Spec.Image != "ghcr.io/openmcp-project/images/platform-service-gateway:v0.0.12" {
		t.Errorf("image = %q, want gateway image", ps.Spec.Image)
	}
}

func TestPlatformService_Dependencies(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := PlatformService("gateway", "img:v1", deployment)
	if len(r.Dependencies) != 1 || r.Dependencies[0] != deployment {
		t.Errorf("expected deployment as sole dependency, got %v", r.Dependencies)
	}
}

func TestPlatformService_ReadyFn(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		generation  int64
		observedGen int64
		phase       string
		wantReady   bool
	}{
		{"ready", 1, 1, commonapi.StatusPhaseReady, true},
		{"progressing", 1, 1, commonapi.StatusPhaseProgressing, false},
		{"generation mismatch", 2, 1, commonapi.StatusPhaseReady, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
			r := PlatformService("gateway", "img:v1", deployment)
			ps := r.Object.(*v1alpha1.PlatformService)
			ps.Name = "gateway"
			ps.Generation = tt.generation
			ps.Status.ObservedGeneration = tt.observedGen
			ps.Status.Phase = tt.phase

			c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(ps).WithObjects(ps).Build()
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(ps), ps); err != nil {
				t.Fatal(err)
			}

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
