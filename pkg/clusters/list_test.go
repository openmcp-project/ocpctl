package clusters

import (
	"fmt"
	"strings"
	"testing"

	testutils "github.com/openmcp-project/ocpctl/pkg/testutils"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
			name: "no kind clusters returns empty map",
			fp:   testutils.FakeProvider{ListResult: []string{}},
		},
		{
			name: "non-platform cluster is ignored",
			fp:   testutils.FakeProvider{ListResult: []string{"some-other-cluster"}},
		},
		{
			name: "client error is skipped with warning",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				ClientErr:  fmt.Errorf("client error"),
			},
		},
		{
			name: "cluster without CRD installed is not managed",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				Client:     &testutils.NoMatchClient{Client: fake.NewClientBuilder().Build()},
			},
		},
		{
			name: "managed environment is returned",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				Client:     testutils.SchemeClient(t),
			},
			wantEnvs: []string{"testenv"},
		},
		{
			name: "clusters in environment are returned",
			fp: testutils.FakeProvider{
				ListResult: []string{"testenv-platform"},
				Client: testutils.SchemeClient(t,
					testutils.ClusterWithKindName(t, "platform", "testenv-platform"),
					testutils.ClusterWithKindName(t, "onboarding", "testenv-onboarding"),
				),
			},
			wantEnvs: []string{"testenv"},
		},
		{
			name: "multiple environments are returned sorted",
			fp: testutils.FakeProvider{
				ListResult: []string{"b-platform", "a-platform"},
				Client:     testutils.SchemeClient(t),
			},
			wantEnvs: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := List(testutils.Ctx(t), &tt.fp)

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
			if len(result) != len(tt.wantEnvs) {
				got := make([]string, 0, len(result))
				for k := range result {
					got = append(got, k)
				}
				t.Fatalf("got environments %v, want %v", got, tt.wantEnvs)
			}
			for _, env := range tt.wantEnvs {
				if _, ok := result[env]; !ok {
					got := make([]string, 0, len(result))
					for k := range result {
						got = append(got, k)
					}
					t.Errorf("expected environment %q in result, got %v", env, got)
				}
			}
		})
	}
}
