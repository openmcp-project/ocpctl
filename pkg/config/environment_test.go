package config

import (
	"testing"
)

func TestMergeFlux(t *testing.T) {
	base := &Environment{
		Spec: EnvironmentSpec{
			Flux: FluxSpec{
				Version:   "v2.3.0",
				Namespace: "flux-system",
				LogLevel:  "info",
			},
		},
	}
	overlay := &Environment{
		Spec: EnvironmentSpec{
			Flux: FluxSpec{
				Version: "v2.4.0",
			},
		},
	}

	result := Merge(base, overlay)

	if result.Spec.Flux.Version != "v2.4.0" {
		t.Errorf("Version: got %q, want %q", result.Spec.Flux.Version, "v2.4.0")
	}
	if result.Spec.Flux.Namespace != "flux-system" {
		t.Errorf("Namespace: got %q, want %q — overlay should not wipe base fields", result.Spec.Flux.Namespace, "flux-system")
	}
	if result.Spec.Flux.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q — overlay should not wipe base fields", result.Spec.Flux.LogLevel, "info")
	}
}

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
