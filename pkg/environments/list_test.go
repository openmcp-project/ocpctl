package environments

import (
	"fmt"
	"strings"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/testutils"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func listClient(t *testing.T) *fake.ClientBuilder {
	t.Helper()
	s := runtime.NewScheme()
	if err := clustersv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(s)
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		fp       testutils.FakeProvider
		wantEnvs []string
		wantErr  string
	}{
		{
			name:    "list clusters error",
			fp:      testutils.FakeProvider{ListErr: fmt.Errorf("list error")},
			wantErr: "listing kind clusters",
		},
		{
			name: "no kind clusters returns empty list",
			fp:   testutils.FakeProvider{ListResult: []string{}},
		},
		{
			name: "non-platform cluster is ignored",
			fp:   testutils.FakeProvider{ListResult: []string{"some-other-cluster"}},
		},
		{
			name: "client error for platform cluster is skipped with warning",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				ClientErr:  fmt.Errorf("client error"),
			},
		},
		{
			name: "platform cluster without cr is skipped",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				Client:     &testutils.NoMatchClient{Client: fake.NewClientBuilder().Build()},
			},
		},
		{
			name: "platform cluster with CRD installed is returned",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				Client:     listClient(t).Build(),
			},
			wantEnvs: []string{"testenv"},
		},
		{
			name: "only platform clusters pass suffix filter",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform", "other-cluster", "testenv2-platform"},
				Client:     listClient(t).Build(),
			},
			wantEnvs: []string{"testenv", "testenv2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envs, err := List(testutils.Ctx(t), &tt.fp)

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
