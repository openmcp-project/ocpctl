package clusters

import (
	"fmt"
	"strings"
	"testing"

	testutils "github.com/openmcp-project/ocpctl/pkg/testutils"
	clustersv1alpha1 "github.com/openmcp-project/openmcp-operator/api/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name        string
		fp          testutils.FakeProvider
		environment string
		cluster     string
		wantConfig  string
		wantErr     string
	}{
		{
			name:        "client error",
			fp:          testutils.FakeProvider{ClientErr: fmt.Errorf("client error")},
			environment: "env",
			cluster:     "platform",
			wantErr:     "could not fetch clusters",
		},
		{
			name: "unmanaged environment",
			fp: testutils.FakeProvider{
				Client: &testutils.NoMatchClient{Client: fake.NewClientBuilder().Build()},
			},
			environment: "env",
			cluster:     "platform",
			wantErr:     "not managed by ocpctl",
		},
		{
			name: "cluster not found",
			fp: testutils.FakeProvider{
				Client: testutils.SchemeClient(t),
			},
			environment: "env",
			cluster:     "nonexistent",
			wantErr:     "not found",
		},
		{
			name: "kubeconfig error",
			fp: testutils.FakeProvider{
				Client:        testutils.SchemeClient(t, testutils.ClusterWithKindName(t, "platform", "env-platform")),
				KubeconfigErr: fmt.Errorf("kubeconfig error"),
			},
			environment: "env",
			cluster:     "platform",
			wantErr:     "kubeconfig error",
		},
		{
			name: "returns kubeconfig",
			fp: testutils.FakeProvider{
				Client:         testutils.SchemeClient(t, testutils.ClusterWithKindName(t, "platform", "env-platform")),
				KubeconfigData: "apiVersion: v1\n",
			},
			environment: "env",
			cluster:     "platform",
			wantConfig:  "apiVersion: v1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := Get(testutils.Ctx(t), tt.environment, tt.cluster, false, &tt.fp)

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
			if config != tt.wantConfig {
				t.Errorf("config = %q, want %q", config, tt.wantConfig)
			}
		})
	}
}

func TestExport(t *testing.T) {
	tests := []struct {
		name        string
		fp          testutils.FakeProvider
		environment string
		cluster     string
		path        string
		wantErr     string
		wantExport  map[string]string
	}{
		{
			name:        "client error",
			fp:          testutils.FakeProvider{ClientErr: fmt.Errorf("client error")},
			environment: "env",
			cluster:     "platform",
			wantErr:     "could not fetch clusters",
		},
		{
			name: "unmanaged environment",
			fp: testutils.FakeProvider{
				Client: &testutils.NoMatchClient{Client: fake.NewClientBuilder().Build()},
			},
			environment: "env",
			cluster:     "platform",
			wantErr:     "not managed by ocpctl",
		},
		{
			name: "cluster not found",
			fp: testutils.FakeProvider{
				Client: testutils.SchemeClient(t),
			},
			environment: "env",
			cluster:     "nonexistent",
			wantErr:     "not found",
		},
		{
			name: "export error",
			fp: testutils.FakeProvider{
				Client:    testutils.SchemeClient(t, testutils.ClusterWithKindName(t, "platform", "env-platform")),
				ExportErr: fmt.Errorf("export error"),
			},
			environment: "env",
			cluster:     "platform",
			wantErr:     "export error",
		},
		{
			name: "exports to path",
			fp: testutils.FakeProvider{
				Client: testutils.SchemeClient(t, testutils.ClusterWithKindName(t, "platform", "env-platform")),
			},
			environment: "env",
			cluster:     "platform",
			path:        "/tmp/kubeconfig",
			wantExport:  map[string]string{"env-platform": "/tmp/kubeconfig"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Export(testutils.Ctx(t), tt.environment, tt.cluster, tt.path, false, &tt.fp)

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
			for _, path := range tt.wantExport {
				if tt.fp.ExportedTo != path {
					t.Errorf("ExportedTo = %q, want %q", tt.fp.ExportedTo, path)
				}
			}
		})
	}
}

func TestGetHandlesEmptyKindName(t *testing.T) {
	cl := &clustersv1alpha1.Cluster{ObjectMeta: metav1.ObjectMeta{Name: "platform"}}
	fp := testutils.FakeProvider{
		Client: testutils.SchemeClient(t, cl),
	}
	_, err := Get(testutils.Ctx(t), "env", "platform", false, &fp)
	if err == nil {
		t.Fatal("expected error for cluster with no provider status, got nil")
	}
}
