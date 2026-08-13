package environments

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/openmcp-project/ocpctl/pkg/config"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	"github.com/openmcp-project/ocpctl/pkg/testutils"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kindImage     = "ghcr.io/openmcp-project/images/cluster-provider-kind:v0.5.0"
	operatorImage = "ghcr.io/openmcp-project/images/openmcp-operator:v1.3.0"
)

func TestBuildPlatformResources(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.Environment
		fp            testutils.FakeProvider
		wantErr       string
		wantResources int
	}{
		{
			name: "no kind cluster provider",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "other-provider", Image: "other-image"}},
				},
			},
			wantErr: "no cluster provider for kind configured",
		},
		{
			name: "empty namespace",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
				},
			},
			wantErr: "namespace must not be empty",
		},
		{
			name: "client error",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
					Operator:         config.OperatorSpec{Image: operatorImage},
				},
			},
			fp:      testutils.FakeProvider{ClientErr: fmt.Errorf("client error")},
			wantErr: "building platform cluster client",
		},
		{
			name: "base only (cluster provider, operator, flux)",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
					Operator:         config.OperatorSpec{Image: operatorImage},
				},
			},
			wantResources: 39,
		},
		{
			name: "base with service provider",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
					Operator:         config.OperatorSpec{Image: operatorImage},
					ServiceProviders: []config.ComponentSpec{
						{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.1.0"},
					},
				},
			},
			wantResources: 40,
		},
		{
			name: "base with platform service",
			cfg: config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
					Operator:         config.OperatorSpec{Image: "operator:v1"},
					PlatformServices: []config.ComponentSpec{
						{Name: "gateway", Image: "ghcr.io/openmcp-project/images/platform-service-gateway:v0.0.14"},
					},
				},
			},
			wantResources: 41,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := buildPlatformResources("test", &tt.cfg, &tt.fp)

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
			got := len(manager.Clusters[0].Resources)
			if got != tt.wantResources {
				t.Errorf("resource count = %d, want %d", got, tt.wantResources)
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name             string
		fp               testutils.FakeProvider
		wantErr          string
		wantEnsureCalled bool
	}{
		{
			name:    "ensure platform cluster error",
			fp:      testutils.FakeProvider{EnsureErr: fmt.Errorf("platform error")},
			wantErr: "ensuring platform cluster",
		},
		{
			name:             "cluster already exists proceeds to apply",
			fp:               testutils.FakeProvider{EnsureCreated: false},
			wantEnsureCalled: true,
		},
		{
			name:             "cluster newly created proceeds to apply",
			fp:               testutils.FakeProvider{EnsureCreated: true},
			wantEnsureCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fp.Client == nil && tt.fp.ClientErr == nil {
				tt.fp.Client = fake.NewClientBuilder().Build()
			}

			log, err := logging.NewLogger(false)
			if err != nil {
				t.Fatal(err)
			}
			// Cancel context to verify EnsurePlatformCluster was called without faking the full apply path
			ctx, cancel := context.WithCancel(logging.IntoContext(context.Background(), log))
			cancel()

			envConfig := config.Environment{
				Spec: config.EnvironmentSpec{
					Namespace:        "test-ns",
					ClusterProviders: []config.ComponentSpec{{Name: "kind", Image: kindImage}},
					Operator:         config.OperatorSpec{Image: operatorImage},
				},
			}

			err = Apply(ctx, "test", &envConfig, &tt.fp)

			if tt.wantEnsureCalled {
				if tt.fp.EnsureCalledWith != "test" {
					t.Errorf("EnsurePlatformCluster called with %q, want %q", tt.fp.EnsureCalledWith, "test")
				}
				if err != context.Canceled {
					t.Errorf("Apply returned %v, want context.Canceled", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("got %q, want error containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
