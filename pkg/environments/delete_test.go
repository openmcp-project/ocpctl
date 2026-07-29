package environments

import (
	"context"
	"fmt"
	"strings"
	"testing"

	kindv1alpha1 "github.com/openmcp-project/cluster-provider-kind/api/v1alpha1"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func clusterWithKindStatus(name, kindClusterName string) *clustersv1alpha1.Cluster {
	cl := &clustersv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: name}}
	ps := kindv1alpha1.ClusterStatus{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterStatus",
			APIVersion: kindv1alpha1.SchemeGroupVersion.String(),
		},
		KindClusterName: kindClusterName,
	}
	if err := cl.Status.SetProviderStatus(ps); err != nil {
		panic(err)
	}
	return cl
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name        string
		fp          fakeProvider
		clusters    []client.Object
		wantErr     string
		wantDeleted []string
	}{
		{
			name:    "client error",
			fp:      fakeProvider{clientErr: fmt.Errorf("client error")},
			wantErr: "connecting to platform cluster",
		},
		{
			name:        "no clusters",
			wantDeleted: nil,
		},
		{
			name: "nil provider status is skipped",
			clusters: []client.Object{
				&clustersv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "no-status-cluster"}},
			},
			wantDeleted: nil,
		},
		{
			name: "single kind cluster is deleted",
			clusters: []client.Object{
				clusterWithKindStatus("onboarding", "onboarding"),
			},
			wantDeleted: []string{"onboarding"},
		},
		{
			name: "platform cluster is deleted last",
			clusters: []client.Object{
				clusterWithKindStatus("platform", "platform"),
				clusterWithKindStatus("onboarding", "onboarding"),
			},
			wantDeleted: []string{"onboarding", "platform"},
		},
		{
			name: "delete error is returned",
			clusters: []client.Object{
				clusterWithKindStatus("onboarding", "onboarding"),
			},
			fp:      fakeProvider{deleteErr: fmt.Errorf("delete error")},
			wantErr: "deleting kind cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fp.clientErr == nil {
				s := runtime.NewScheme()
				if err := clustersv1alpha1.AddToScheme(s); err != nil {
					t.Fatal(err)
				}
				builder := fake.NewClientBuilder().WithScheme(s)
				if len(tt.clusters) > 0 {
					builder = builder.WithStatusSubresource(tt.clusters...).WithObjects(tt.clusters...)
				}
				tt.fp.client = builder.Build()
			}

			log, err := logging.NewLogger(false)
			if err != nil {
				t.Fatal(err)
			}
			ctx := logging.IntoContext(context.Background(), log)
			err = Delete(ctx, "env", &tt.fp)

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
			if len(tt.fp.deletedClusters) != len(tt.wantDeleted) {
				t.Fatalf("deleted %v, want %v", tt.fp.deletedClusters, tt.wantDeleted)
			}
			for i := range tt.fp.deletedClusters {
				if tt.fp.deletedClusters[i] != tt.wantDeleted[i] {
					t.Errorf("deleted = %q, want %q", tt.fp.deletedClusters[i], tt.wantDeleted[i])
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
