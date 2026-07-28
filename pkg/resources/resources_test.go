package resources

import (
	"context"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplySummary_Total(t *testing.T) {
	tests := []struct {
		name    string
		applied int
		waiting int
		want    int
	}{
		{"all applied", 3, 0, 3},
		{"all waiting", 0, 2, 2},
		{"mixed", 2, 3, 5},
		{"empty", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := ApplySummary{
				Applied:        make([]*Resource, tt.applied),
				WaitingForDeps: make([]*Resource, tt.waiting),
			}
			if got := s.Total(); got != tt.want {
				t.Errorf("Total() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResource_String(t *testing.T) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"}}
	r := &Resource{Object: cm}
	got := r.String()
	want := "ConfigMap/test-ns/test-cm"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestManager_Apply(t *testing.T) {
	tests := []struct {
		name        string
		resources   []*Resource
		wantApplied int
		wantWaiting int
		wantReady   int
		verify      func(*testing.T, *Manager)
	}{
		{
			name:        "creates resource",
			resources:   []*Resource{{Object: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}}},
			wantApplied: 1,
			wantReady:   1,
		},
		{
			name: "waits for dependency",
			resources: func() []*Resource {
				parent := &Resource{
					Object:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}},
					ReadyFn: func(_ context.Context) (bool, error) { return false, nil },
				}
				child := &Resource{
					Object:       &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-cm"}},
					Dependencies: []client.Object{parent.Object},
				}
				return []*Resource{parent, child}
			}(),
			wantApplied: 1,
			wantWaiting: 1,
		},
		{
			name: "resource not ready",
			resources: []*Resource{{
				Object:  &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}},
				ReadyFn: func(_ context.Context) (bool, error) { return false, nil },
			}},
			wantApplied: 1,
		},
		{
			name: "resources is mutated",
			resources: func() []*Resource {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}
				return []*Resource{{
					Object: ns,
					MutateFn: func(_ context.Context) error {
						ns.Labels = map[string]string{"mutated": "true"}
						return nil
					},
				}}
			}(),
			wantApplied: 1,
			wantReady:   1,
			verify: func(t *testing.T, m *Manager) {
				ns := m.Clusters[0].Resources[0].Object.(*corev1.Namespace)
				if ns.Labels["mutated"] != "true" {
					t.Errorf("label mutated = %q, want %q", ns.Labels["mutated"], "true")
				}
			},
		},
		{
			name:      "skips external resource",
			resources: []*Resource{{Object: &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns"}}, External: true}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{}
			c := fake.NewClientBuilder().Build()
			m.AddClusters(&Cluster{Client: c, Resources: tt.resources})
			log, err := logging.NewLogger(false)
			if err != nil {
				t.Fatal(err)
			}
			summary, err := m.Apply(logging.IntoContext(context.Background(), log))
			if err != nil {
				t.Fatalf("Apply error: %v", err)
			}
			if len(summary.Applied) != tt.wantApplied {
				t.Errorf("Applied = %d, want %d", len(summary.Applied), tt.wantApplied)
			}
			if len(summary.WaitingForDeps) != tt.wantWaiting {
				t.Errorf("WaitingForDeps = %d, want %d", len(summary.WaitingForDeps), tt.wantWaiting)
			}
			if len(summary.Ready) != tt.wantReady {
				t.Errorf("Ready = %d, want %d", len(summary.Ready), tt.wantReady)
			}
			if tt.verify != nil {
				tt.verify(t, m)
			}
		})
	}
}
