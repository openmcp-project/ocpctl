package platform

import (
	"context"
	"fmt"
	"strings"

	fluxinstall "github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/openmcp-project/ocpctl/pkg/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)


type FluxCD struct {
	// Namespace is the Kubernetes namespace where FluxCD components will be installed.
	// If empty, defaults to "flux-system".
	Namespace string
}

func (f *FluxCD) PlatformFlux() ([]*resources.Resource, error) {
	// Use configured namespace or default to "flux-system"
	namespace := f.Namespace
	if namespace == "" {
		namespace = "flux-system"
	}

	// Generate the Flux installation manifests
	options := fluxinstall.MakeDefaultOptions()
	options.Namespace = namespace
	options.Components = []string{
		"source-controller",
		"kustomize-controller",
		"helm-controller",
		"notification-controller",
	}
	manifest, err := fluxinstall.Generate(options, "")

	if err != nil {
		return nil, fmt.Errorf("failed to generate flux manifests: %w", err)
	}
	var res []*resources.Resource
	for m := range strings.SplitSeq(manifest.Content, "---\n") {
		if strings.TrimSpace(m) == "" {
			continue
		}

		desired := &unstructured.Unstructured{}
		if err := yaml.Unmarshal([]byte(m), &desired.Object); err != nil {
			return nil, fmt.Errorf("failed to unmarshal desired manifest: %w", err)
		}

		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(desired.GroupVersionKind())
		obj.SetName(desired.GetName())
		obj.SetNamespace(desired.GetNamespace())

		res = append(res, &resources.Resource{
			Object: obj,
			MutateFn: func(_ context.Context) error {
				serverOwned := map[string]bool{
					"apiVersion": true, "kind": true,
					"metadata": true, "status": true,
				}
				for k, v := range desired.Object {
					if !serverOwned[k] {
						obj.Object[k] = v
					}
				}
				obj.SetLabels(desired.GetLabels())
				obj.SetAnnotations(desired.GetAnnotations())
				return nil
			},
		})
	}
	return res, nil
}
