package platform

import (
	"context"
	"testing"

	fluxinstall "github.com/fluxcd/flux2/v2/pkg/manifestgen/install"
	"github.com/openmcp-project/ocpctl/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBuildFluxOptions(t *testing.T) {
	defaults := fluxinstall.MakeDefaultOptions()

	tests := []struct {
		name      string
		options   *config.FluxSpec
		checkFunc func(t *testing.T, got fluxinstall.Options)
	}{
		{
			name:    "nil options returns defaults",
			options: nil,
			checkFunc: func(t *testing.T, got fluxinstall.Options) {
				if got.Namespace != defaults.Namespace {
					t.Errorf("namespace = %q, want %q", got.Namespace, defaults.Namespace)
				}
				if got.Registry != defaults.Registry {
					t.Errorf("registry = %q, want %q", got.Registry, defaults.Registry)
				}
				if got.LogLevel != defaults.LogLevel {
					t.Errorf("logLevel = %q, want %q", got.LogLevel, defaults.LogLevel)
				}
			},
		},
		{
			name:    "empty options struct preserves defaults",
			options: &config.FluxSpec{},
			checkFunc: func(t *testing.T, got fluxinstall.Options) {
				if got.Namespace != defaults.Namespace {
					t.Errorf("namespace = %q, want %q", got.Namespace, defaults.Namespace)
				}
				if got.Registry != defaults.Registry {
					t.Errorf("registry = %q, want %q", got.Registry, defaults.Registry)
				}
				if got.LogLevel != defaults.LogLevel {
					t.Errorf("logLevel = %q, want %q", got.LogLevel, defaults.LogLevel)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildFluxOptions(tt.options)
			tt.checkFunc(t, got)
		})
	}
}

func TestPlatformFlux(t *testing.T) {
	tests := []struct {
		name      string
		checkFunc func(t *testing.T, obj *unstructured.Unstructured)
	}{
		{
			name: "all resources have a name",
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetName() == "" {
					t.Errorf("resource %s/%s has empty name", obj.GetKind(), obj.GetNamespace())
				}
			},
		},
		{
			name: "all resources have a kind",
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetKind() == "" {
					t.Errorf("resource %q has empty kind", obj.GetName())
				}
			},
		},
		{
			name: "all resources have an apiVersion",
			checkFunc: func(t *testing.T, obj *unstructured.Unstructured) {
				if obj.GetAPIVersion() == "" {
					t.Errorf("resource %s/%s has empty apiVersion", obj.GetKind(), obj.GetName())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources, err := PlatformFlux(nil)
			if err != nil {
				t.Fatalf("PlatformFlux error: %v", err)
			}
			if len(resources) == 0 {
				t.Fatal("expected at least one resource, got none")
			}
			for _, r := range resources {
				if err := r.MutateFn(context.Background()); err != nil {
					t.Fatalf("MutateFn error for %s: %v", r.Object.GetName(), err)
				}
				obj := r.Object.(*unstructured.Unstructured)
				tt.checkFunc(t, obj)
			}
		})
	}
}

func TestPlatformFlux_MutateFn_CopiesFields(t *testing.T) {
	fluxResources, err := PlatformFlux(nil)
	if err != nil {
		t.Fatalf("PlatformFlux error: %v", err)
	}
	for _, r := range fluxResources {
		obj := r.Object.(*unstructured.Unstructured)
		if err := r.MutateFn(context.Background()); err != nil {
			t.Fatalf("MutateFn error for %s: %v", obj.GetName(), err)
		}
		kind := obj.GetKind()
		if kind == "ServiceAccount" || kind == "Namespace" {
			continue
		}
		hasContent := false
		for k := range obj.Object {
			if !skipFields[k] {
				hasContent = true
				break
			}
		}
		if !hasContent {
			t.Errorf("%s/%s: MutateFn left no non-skipped fields", kind, obj.GetName())
		}
	}
}

func TestPlatformFlux_MutateFn_NoOverwriteSkippedFields(t *testing.T) {
	fluxResources, err := PlatformFlux(nil)
	if err != nil {
		t.Fatalf("PlatformFlux error: %v", err)
	}
	for _, r := range fluxResources {
		obj := r.Object.(*unstructured.Unstructured)
		obj.Object["status"] = map[string]any{"test-status": "do-not-overwrite"}
		if err := r.MutateFn(context.Background()); err != nil {
			t.Fatalf("MutateFn error for %s: %v", obj.GetName(), err)
		}
		status, _, _ := unstructured.NestedString(obj.Object, "status", "test-status")
		if status != "do-not-overwrite" {
			t.Errorf("%s/%s: status field: want %q, have %q", obj.GetKind(), obj.GetName(), "do-not-overwrite", status)
		}
	}
}

func TestGetReadyFn(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		spec      map[string]any
		status    map[string]any
		wantReady bool
		wantNil   bool
		wantErr   bool
	}{
		{
			name:      "readyReplicas equals replicas",
			kind:      "Deployment",
			spec:      map[string]any{"replicas": int64(1)},
			status:    map[string]any{"readyReplicas": int64(1)},
			wantReady: true,
		},
		{
			name:      "readyReplicas less than replicas",
			kind:      "Deployment",
			spec:      map[string]any{"replicas": int64(2)},
			status:    map[string]any{"readyReplicas": int64(1)},
			wantReady: false,
		},
		{
			name:      "nil replicas defaults to 1",
			kind:      "Deployment",
			spec:      map[string]any{},
			status:    map[string]any{"readyReplicas": int64(1)},
			wantReady: true,
		},
		{
			name:    "not kind deployment returns nil",
			kind:    "NotDeployment",
			wantNil: true,
		},
		{
			name:    "invalid spec triggers error",
			kind:    "Deployment",
			spec:    map[string]any{"replicas": "not-a-number"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &unstructured.Unstructured{Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       tt.kind,
				"spec":       tt.spec,
				"status":     tt.status,
			}}

			fn := getReadyFn(obj)
			if tt.wantNil {
				if fn != nil {
					t.Error("ready fn: want nil, have non-nil")
				}
				return
			}
			if fn == nil {
				t.Fatal("ready fn: want non-nil, have nil")
			}
			got, err := fn(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Error("error: want err, have nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadyFn error: %v", err)
			}
			if got != tt.wantReady {
				t.Errorf("ready: want %v, have %v", tt.wantReady, got)
			}
		})
	}
}
