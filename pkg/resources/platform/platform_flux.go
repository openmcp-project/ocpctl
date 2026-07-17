package platform

import (
	"context"
	"fmt"
	"strings"

	fluxinstall "github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/openmcp-project/ocpctl/pkg/config"
	"github.com/openmcp-project/ocpctl/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var skipFields = map[string]bool{
	"apiVersion": true,
	"kind":       true,
	"metadata":   true,
	"status":     true,
}

func PlatformFlux(opts *config.FluxSpec) ([]*resources.Resource, error) {
	options := buildFluxOptions(opts)
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
			Object:  obj,
			ReadyFn: getReadyFn(obj),
			MutateFn: func(_ context.Context) error {
				desired := desired.DeepCopy()
				for k, v := range desired.Object {
					if !skipFields[k] {
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

func getReadyFn(obj *unstructured.Unstructured) resources.ReadyFn {
	gvk := obj.GroupVersionKind()
	if gvk.Group != "apps" || gvk.Kind != "Deployment" {
		return nil
	}
	return func(_ context.Context) (bool, error) {
		deployment := &appsv1.Deployment{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, deployment); err != nil {
			return false, err
		}
		if deployment.Generation != deployment.Status.ObservedGeneration {
			return false, nil
		}
		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}
		return deployment.Status.ReadyReplicas >= desired, nil
	}
}

func buildFluxOptions(opts *config.FluxSpec) fluxinstall.Options {
	options := fluxinstall.MakeDefaultOptions()
	options.ComponentsExtra = []string{}
	if opts == nil {
		return options
	}
	if opts.Version != "" {
		options.Version = opts.Version
	}
	if opts.Namespace != "" {
		options.Namespace = opts.Namespace
	}
	if opts.Registry != "" {
		options.Registry = opts.Registry
	}
	if opts.RegistryCredential != "" {
		options.RegistryCredential = opts.RegistryCredential
	}
	if opts.ImagePullSecret != "" {
		options.ImagePullSecret = opts.ImagePullSecret
	}
	if opts.BaseURL != "" {
		options.BaseURL = opts.BaseURL
	}
	if opts.LogLevel != "" {
		options.LogLevel = opts.LogLevel
	}
	if opts.ComponentsExtra != nil {
		options.ComponentsExtra = opts.ComponentsExtra
	}
	if opts.ClusterDomain != "" {
		options.ClusterDomain = opts.ClusterDomain
	}
	if opts.NetworkPolicy != nil {
		options.NetworkPolicy = *opts.NetworkPolicy
	}
	if opts.TolerationKeys != nil {
		options.TolerationKeys = opts.TolerationKeys
	}
	return options
}
