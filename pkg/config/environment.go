package config

import (
	_ "embed"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

//go:embed environment-defaults.yaml
var defaultConfigData []byte

func Default() *Environment {
	var env Environment
	if err := yaml.Unmarshal(defaultConfigData, &env); err != nil {
		panic(fmt.Sprintf("parsing embedded default config: %v", err))
	}
	return &env
}

func Load(path string) (*Environment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var env Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return &env, nil
}

// Merge returns a new Environment with overlay values applied on top of base.
// Non-empty string fields from overlay take precedence. Component slices are
// merged by name: overlay entries update matching base entries or are appended.
func Merge(base, overlay *Environment) *Environment {
	result := *base
	if overlay.Spec.Namespace != "" {
		result.Spec.Namespace = overlay.Spec.Namespace
	}
	if overlay.Spec.Operator.Image != "" {
		result.Spec.Operator.Image = overlay.Spec.Operator.Image
	}
	result.Spec.Flux = mergeFlux(base.Spec.Flux, overlay.Spec.Flux)
	result.Spec.ClusterProviders = mergeComponents(base.Spec.ClusterProviders, overlay.Spec.ClusterProviders)
	result.Spec.ServiceProviders = mergeComponents(base.Spec.ServiceProviders, overlay.Spec.ServiceProviders)
	result.Spec.PlatformServices = mergeComponents(base.Spec.PlatformServices, overlay.Spec.PlatformServices)
	return &result
}

func mergeComponents(base, overlay []ComponentSpec) []ComponentSpec {
	if len(overlay) == 0 {
		return base
	}
	result := make([]ComponentSpec, len(base))
	copy(result, base)
	for _, o := range overlay {
		found := false
		for i, b := range result {
			if b.Name == o.Name {
				if o.Image != "" {
					result[i].Image = o.Image
				}
				found = true
				break
			}
		}
		if !found {
			result = append(result, o)
		}
	}
	return result
}

func mergeFlux(base, overlay FluxSpec) FluxSpec {
	result := base
	if overlay.Version != "" {
		result.Version = overlay.Version
	}
	if overlay.Namespace != "" {
		result.Namespace = overlay.Namespace
	}
	if overlay.Registry != "" {
		result.Registry = overlay.Registry
	}
	if overlay.RegistryCredential != "" {
		result.RegistryCredential = overlay.RegistryCredential
	}
	if overlay.ImagePullSecret != "" {
		result.ImagePullSecret = overlay.ImagePullSecret
	}
	if overlay.BaseURL != "" {
		result.BaseURL = overlay.BaseURL
	}
	if overlay.LogLevel != "" {
		result.LogLevel = overlay.LogLevel
	}
	if overlay.ComponentsExtra != nil {
		result.ComponentsExtra = overlay.ComponentsExtra
	}
	if overlay.ClusterDomain != "" {
		result.ClusterDomain = overlay.ClusterDomain
	}
	if overlay.NetworkPolicy != nil {
		result.NetworkPolicy = overlay.NetworkPolicy
	}
	if overlay.TolerationKeys != nil {
		result.TolerationKeys = overlay.TolerationKeys
	}
	return result
}

type Environment struct {
	APIVersion string          `json:"apiVersion"`
	Kind       string          `json:"kind"`
	Spec       EnvironmentSpec `json:"spec"`
}

type EnvironmentSpec struct {
	Namespace        string          `json:"namespace"`
	Operator         OperatorSpec    `json:"operator"`
	ClusterProviders []ComponentSpec `json:"clusterProviders"`
	ServiceProviders []ComponentSpec `json:"serviceProviders"`
	PlatformServices []ComponentSpec `json:"platformServices"`
	Flux             FluxSpec        `json:"flux"`
}

type OperatorSpec struct {
	Image string `json:"image"`
}

type ComponentSpec struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type FluxSpec struct {
	Version            string   `json:"version"`
	Namespace          string   `json:"namespace"`
	Registry           string   `json:"registry"`
	RegistryCredential string   `json:"registryCredential"`
	ImagePullSecret    string   `json:"imagePullSecret"`
	BaseURL            string   `json:"baseURL"`
	LogLevel           string   `json:"logLevel"`
	ComponentsExtra    []string `json:"componentsExtra"`
	ClusterDomain      string   `json:"clusterDomain"`
	NetworkPolicy      *bool    `json:"networkPolicy"`
	TolerationKeys     []string `json:"tolerationKeys"`
}
