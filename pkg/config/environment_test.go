package config

import (
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
