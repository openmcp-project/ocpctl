package environments

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// noMatchClient wraps a fake client and returns a NoKindMatchError for List calls.
type noMatchClient struct {
	client.Client
}

func (n *noMatchClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return &apimeta.NoKindMatchError{GroupKind: schema.GroupKind{Group: "clusters.openmcp.cloud", Kind: "ClusterList"}}
}

func listClient(t *testing.T) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	if err := clustersv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(s).Build()
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		fp       fakeProvider
		wantEnvs []string
		wantErr  string
	}{
		{
			name:    "list clusters error",
			fp:      fakeProvider{listErr: fmt.Errorf("list error")},
			wantErr: "listing kind clusters",
		},
		{
			name: "no kind clusters returns empty list",
			fp:   fakeProvider{listClusters: []string{}},
		},
		{
			name: "non-platform cluster is ignored",
			fp:   fakeProvider{listClusters: []string{"some-other-cluster"}},
		},
		{
			name: "client error for platform cluster is skipped with warning",
			fp: fakeProvider{
				listClusters: []string{"testenv-platform"},
				clientErr:    fmt.Errorf("client error"),
			},
		},
		{
			name: "platform cluster without cr is skipped",
			fp: fakeProvider{
				listClusters: []string{"testenv-platform"},
				client:       &noMatchClient{fake.NewClientBuilder().Build()},
			},
		},
		{
			name: "platform cluster with CRD installed is returned",
			fp: fakeProvider{
				listClusters: []string{"testenv-platform"},
				client:       listClient(t),
			},
			wantEnvs: []string{"testenv"},
		},
		{
			name: "only platform clusters pass suffix filter",
			fp: fakeProvider{
				listClusters: []string{"testenv-platform", "other-cluster", "testenv2-platform"},
				client:       listClient(t),
			},
			wantEnvs: []string{"testenv", "testenv2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, err := logging.NewLogger(false)
			if err != nil {
				t.Fatal(err)
			}
			ctx := logging.IntoContext(context.Background(), log)
			envs, err := List(ctx, &tt.fp)

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
			if len(envs) != len(tt.wantEnvs) {
				t.Fatalf("envs = %v, want %v", envs, tt.wantEnvs)
			}
			for i := range envs {
				if envs[i] != tt.wantEnvs[i] {
					t.Errorf("envs[%d] = %q, want %q", i, envs[i], tt.wantEnvs[i])
				}
			}
		})
	}
}
