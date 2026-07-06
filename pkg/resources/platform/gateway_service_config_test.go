package platform

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGatewayServiceConfig_MutateFn(t *testing.T) {
	r := GatewayServiceConfig("gateway")

	if err := r.MutateFn(context.Background()); err != nil {
		t.Fatalf("MutateFn error: %v", err)
	}

	obj := r.Object.(*unstructured.Unstructured)

	baseDomain, _, _ := unstructured.NestedString(obj.Object, "spec", "dns", "baseDomain")
	if baseDomain == "" {
		t.Errorf("spec.dns.baseDomain is empty")
	}

	chartTag, _, _ := unstructured.NestedString(obj.Object, "spec", "envoyGateway", "chart", "tag")
	if chartTag == "" {
		t.Errorf("spec.envoyGateway.chart.tag is empty")
	}
}

func TestGatewayServiceConfig_NoDependencies(t *testing.T) {
	r := GatewayServiceConfig("gateway")
	if len(r.Dependencies) != 0 {
		t.Errorf("expected no dependencies, got %v", r.Dependencies)
	}
}

func TestGatewayServiceConfig_GVK(t *testing.T) {
	r := GatewayServiceConfig("gateway")

	obj := r.Object.(*unstructured.Unstructured)
	gvk := obj.GroupVersionKind()
	if gvk.Group != "gateway.openmcp.cloud" || gvk.Version != "v1alpha1" || gvk.Kind != "GatewayServiceConfig" {
		t.Errorf("unexpected GVK: %v", gvk)
	}
	if obj.GetName() != "gateway" {
		t.Errorf("name = %q, want %q", obj.GetName(), "gateway")
	}
}
