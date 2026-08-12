package environments

import (
	"fmt"
	"strings"
	"testing"

	testutils "github.com/openmcp-project/ocpctl/pkg/testutils"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDelete(t *testing.T) {
	tests := []struct {
		name        string
		fp          testutils.FakeProvider
		clusters    []client.Object
		wantErr     string
		wantDeleted []string
	}{
		{
			name:    "client error",
			fp:      testutils.FakeProvider{ClientErr: fmt.Errorf("client error")},
			wantErr: "connecting to platform cluster",
		},
		{
			name:        "no clusters",
			wantDeleted: nil,
		},
		{
			name: "nil provider status is skipped",
			clusters: []client.Object{
				&clustersv1alpha1.Cluster{},
			},
			wantDeleted: nil,
		},
		{
			name: "single kind cluster is deleted",
			clusters: []client.Object{
				testutils.ClusterWithKindName(t, "onboarding", "onboarding"),
			},
			wantDeleted: []string{"onboarding"},
		},
		{
			name: "platform cluster is deleted last",
			clusters: []client.Object{
				testutils.ClusterWithKindName(t, "platform", "platform"),
				testutils.ClusterWithKindName(t, "onboarding", "onboarding"),
			},
			wantDeleted: []string{"onboarding", "platform"},
		},
		{
			name: "delete error is returned",
			clusters: []client.Object{
				testutils.ClusterWithKindName(t, "onboarding", "onboarding"),
			},
			fp:      testutils.FakeProvider{DeleteErr: fmt.Errorf("delete error")},
			wantErr: "deleting kind cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fp.Client == nil && tt.fp.ClientErr == nil {
				s := runtime.NewScheme()
				if err := clustersv1alpha1.AddToScheme(s); err != nil {
					t.Fatal(err)
				}
				builder := fake.NewClientBuilder().WithScheme(s)
				if len(tt.clusters) > 0 {
					builder = builder.WithStatusSubresource(tt.clusters...).WithObjects(tt.clusters...)
				}
				tt.fp.Client = builder.Build()
			}

			err := Delete(testutils.Ctx(t), "env", &tt.fp)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("got %q, want error containing %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tt.fp.DeletedClusters) != len(tt.wantDeleted) {
				t.Fatalf("deleted %v, want %v", tt.fp.DeletedClusters, tt.wantDeleted)
			}
			for i := range tt.fp.DeletedClusters {
				if tt.fp.DeletedClusters[i] != tt.wantDeleted[i] {
					t.Errorf("deleted = %q, want %q", tt.fp.DeletedClusters[i], tt.wantDeleted[i])
				}
			}
		})
	}
}

func TestSortPlatformLast(t *testing.T) {
	tests := []struct {
		name            string
		clusters        []string
		platformCluster string
		want            []string
	}{
		{
			name:            "platform in the middle",
			clusters:        []string{"test-onboarding", "test-platform", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1", "test-platform"},
		},
		{
			name:            "platform already last",
			clusters:        []string{"test-onboarding", "test-worker-1", "test-platform"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1", "test-platform"},
		},
		{
			name:            "platform first",
			clusters:        []string{"test-platform", "test-onboarding", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-worker-1", "test-onboarding", "test-platform"},
		},
		{
			name:            "only platform",
			clusters:        []string{"test-platform"},
			platformCluster: "test-platform",
			want:            []string{"test-platform"},
		},
		{
			name:            "platform not present",
			clusters:        []string{"test-onboarding", "test-worker-1"},
			platformCluster: "test-platform",
			want:            []string{"test-onboarding", "test-worker-1"},
		},
		{
			name:            "empty slice",
			clusters:        []string{},
			platformCluster: "test-platform",
			want:            []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortPlatformLast(tt.clusters, tt.platformCluster)
			if len(tt.clusters) != len(tt.want) {
				t.Fatalf("got %v, want %v", tt.clusters, tt.want)
			}
			for i := range tt.clusters {
				if tt.clusters[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q (full slice: %v)", i, tt.clusters[i], tt.want[i], tt.clusters)
				}
			}
		})
	}
}
