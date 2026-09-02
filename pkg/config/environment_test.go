package config

import (
	"os"
	"testing"
)

func TestDefaultContainsExpectedComponents(t *testing.T) {
	cfg := Default()

	wantServiceProviders := []string{"crossplane", "flux", "ocm", "kro"}
	wantPlatformServices := []string{"gateway"}

	spByName := make(map[string]string)
	for _, sp := range cfg.Spec.ServiceProviders {
		spByName[sp.Name] = sp.Image
	}
	for _, name := range wantServiceProviders {
		img, ok := spByName[name]
		if !ok {
			t.Errorf("serviceProvider %q missing from defaults", name)
			continue
		}
		if img == "" {
			t.Errorf("serviceProvider %q has empty image", name)
		}
	}

	psByName := make(map[string]string)
	for _, ps := range cfg.Spec.PlatformServices {
		psByName[ps.Name] = ps.Image
	}
	for _, name := range wantPlatformServices {
		img, ok := psByName[name]
		if !ok {
			t.Errorf("platformService %q missing from defaults", name)
			continue
		}
		if img == "" {
			t.Errorf("platformService %q has empty image", name)
		}
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		valFunc func(*testing.T, *Environment)
	}{
		{
			name: "valid config",
			content: `apiVersion: ocpctl.open-control-plane.io/v1alpha1
kind: Environment
spec:
  namespace: my-ns
  serviceProviders:
    - name: flux
      image: ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0
`,
			valFunc: func(t *testing.T, e *Environment) {
				if e.Spec.Namespace != "my-ns" {
					t.Errorf("namespace = %q, want %q", e.Spec.Namespace, "my-ns")
				}
				if len(e.Spec.ServiceProviders) != 1 || e.Spec.ServiceProviders[0].Name != "flux" {
					t.Errorf("unexpected service providers: %v", e.Spec.ServiceProviders)
				}
			},
		},
		{
			name:    "file not found",
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			content: "not-valid-yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "nonexistent.yaml"
			if tt.content != "" {
				f, err := os.CreateTemp(t.TempDir(), "*.yaml")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(f.Name(), []byte(tt.content), 0600); err != nil {
					t.Fatal(err)
				}
				path = f.Name()
			}
			got, err := Load(path)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.valFunc != nil && got != nil {
				tt.valFunc(t, got)
			}
		})
	}
}

func TestMergeComponents(t *testing.T) {
	tests := []struct {
		name    string
		base    []ComponentSpec
		overlay []ComponentSpec
		mode    MergeMode
		want    []ComponentSpec
	}{
		{
			name: "empty overlay returns base",
			base: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
			overlay: []ComponentSpec{},
			want: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
		},
		{
			name: "overlay overrides base for same component",
			base: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
			overlay: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.1.0"},
			},
			want: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.1.0"},
			},
		},
		{
			name: "empty image in overlay does not override base",
			base: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
			overlay: []ComponentSpec{
				{Name: "sp-flux", Image: ""},
			},
			want: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
		},
		{
			name: "additional component in overlay added to based",
			base: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
			},
			overlay: []ComponentSpec{
				{Name: "sp-crossplane", Image: "ghcr.io/openmcp-project/images/service-provider-crossplane:v1.0.0"},
			},
			want: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
				{Name: "sp-crossplane", Image: "ghcr.io/openmcp-project/images/service-provider-crossplane:v1.0.0"},
			},
		},
		{
			name: "multiple base components overlay overrides one and adds one",
			base: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
				{Name: "sp-crossplane", Image: "ghcr.io/openmcp-project/images/service-provider-crossplane:v1.0.0"},
				{Name: "sp-ocm", Image: "ghcr.io/openmcp-project/images/service-provider-ocm:v1.0.0"},
			},
			overlay: []ComponentSpec{
				{Name: "sp-crossplane", Image: "ghcr.io/openmcp-project/images/service-provider-crossplane:v2.0.0"},
				{Name: "sp-kro", Image: "ghcr.io/openmcp-project/images/service-provider-kro:v1.0.0"},
			},
			want: []ComponentSpec{
				{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"},
				{Name: "sp-crossplane", Image: "ghcr.io/openmcp-project/images/service-provider-crossplane:v2.0.0"},
				{Name: "sp-ocm", Image: "ghcr.io/openmcp-project/images/service-provider-ocm:v1.0.0"},
				{Name: "sp-kro", Image: "ghcr.io/openmcp-project/images/service-provider-kro:v1.0.0"},
			},
		},
		{
			name:    "nil overlay returns base",
			base:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			overlay: nil,
			want:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
		},
		{
			name:    "nil base with overlay returns overlay",
			base:    nil,
			overlay: []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			want:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
		},
		{
			name:    "both nil returns nil",
			base:    nil,
			overlay: nil,
			want:    nil,
		},
		{
			name:    "replace mode with empty overlay returns empty",
			base:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			overlay: []ComponentSpec{},
			mode:    MergeModeReplace,
			want:    nil,
		},
		{
			name:    "replace mode with nil overlay returns nil",
			base:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			overlay: nil,
			mode:    MergeModeReplace,
			want:    nil,
		},
		{
			name:    "replace mode with overlay wins wholesale",
			base:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			overlay: []ComponentSpec{{Name: "sp-kro", Image: "ghcr.io/openmcp-project/images/service-provider-kro:v1.0.0"}},
			mode:    MergeModeReplace,
			want:    []ComponentSpec{{Name: "sp-kro", Image: "ghcr.io/openmcp-project/images/service-provider-kro:v1.0.0"}},
		},
		{
			name:    "explicit merge mode behaves like default",
			base:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
			overlay: []ComponentSpec{},
			mode:    MergeModeMerge,
			want:    []ComponentSpec{{Name: "sp-flux", Image: "ghcr.io/openmcp-project/images/service-provider-flux:v1.0.0"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeComponents(tt.base, tt.overlay, tt.mode)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMergeServiceProvidersReplace(t *testing.T) {
	base := Default()
	if len(base.Spec.ServiceProviders) == 0 {
		t.Fatal("expected defaults to ship service providers")
	}

	overlay := &Environment{Spec: EnvironmentSpec{ServiceProvidersMode: MergeModeReplace}}
	got := Merge(base, overlay)
	if len(got.Spec.ServiceProviders) != 0 {
		t.Errorf("replace mode with empty overlay: got %d service providers, want 0", len(got.Spec.ServiceProviders))
	}
	// Other sections keep defaults.
	if len(got.Spec.PlatformServices) != len(base.Spec.PlatformServices) {
		t.Errorf("platform services changed: got %d, want %d", len(got.Spec.PlatformServices), len(base.Spec.PlatformServices))
	}
}

func TestMergeDefaultsRegression(t *testing.T) {
	base := Default()
	overlay := &Environment{Spec: EnvironmentSpec{Namespace: "my-ns"}}
	got := Merge(base, overlay)
	if len(got.Spec.ServiceProviders) != len(base.Spec.ServiceProviders) {
		t.Errorf("omitted mode changed service providers: got %d, want %d", len(got.Spec.ServiceProviders), len(base.Spec.ServiceProviders))
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mode    MergeMode
		wantErr bool
	}{
		{name: "empty mode ok", mode: "", wantErr: false},
		{name: "merge ok", mode: MergeModeMerge, wantErr: false},
		{name: "replace ok", mode: MergeModeReplace, wantErr: false},
		{name: "invalid mode", mode: "bogus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Environment{Spec: EnvironmentSpec{ServiceProvidersMode: tt.mode}}
			err := e.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
