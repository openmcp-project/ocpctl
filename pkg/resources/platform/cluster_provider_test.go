package platform

import (
	"context"
	"testing"

	commonapi "github.com/openmcp-project/openmcp-operator/api/common"
	"github.com/openmcp-project/openmcp-operator/api/provider/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cpImage = "ghcr.io/openmcp-project/images/cluster-provider-kind:v0.5.0"
)

func TestClusterProvider_Dependencies(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := ClusterProvider(cpImage, deployment)

	if len(r.Dependencies) != 1 || r.Dependencies[0] != deployment {
		t.Errorf("expected deployment as sole dependency, got %v", r.Dependencies)
	}
}

func TestClusterProvider_MutateFn(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := ClusterProvider(cpImage, deployment)

	provider := r.Object.(*v1alpha1.ClusterProvider)
	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	if provider.Name != "kind" {
		t.Errorf("name = %q, want %q", provider.Name, "kind")
	}

	if provider.Spec.Image != cpImage {
		t.Errorf("image = %q, want %q", provider.Spec.Image, cpImage)
	}

	volumes := provider.Spec.ExtraVolumes
	if len(volumes) != 1 || volumes[0].Name != "docker-socket" {
		t.Errorf("unexpected extra volumes: %v", volumes)
	}
	if volumes[0].HostPath == nil || volumes[0].HostPath.Path != "/var/run/host-docker.sock" {
		t.Errorf("unexpected docker-socket hostPath: %v", volumes[0].HostPath)
	}
	if volumes[0].HostPath.Type == nil || *volumes[0].HostPath.Type != corev1.HostPathSocket {
		t.Errorf("hostPath type = %v, want HostPathSocket", volumes[0].HostPath.Type)
	}

	mounts := provider.Spec.ExtraVolumeMounts
	if len(mounts) != 1 || mounts[0].Name != "docker-socket" {
		t.Errorf("unexpected extra volume mounts: %v", mounts)
	}
	if mounts[0].MountPath != "/var/run/docker.sock" {
		t.Errorf("mount path = %q, want %q", mounts[0].MountPath, "/var/run/docker.sock")
	}
}

func TestClusterProvider_ReadyFn(t *testing.T) {
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
			r := ClusterProvider(cpImage, deployment)
			provider := r.Object.(*v1alpha1.ClusterProvider)
			provider.Generation = tt.generation
			provider.Status.ObservedGeneration = tt.observedGen
			provider.Status.Phase = tt.phase

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
