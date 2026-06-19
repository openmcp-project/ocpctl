package platform

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func testGatewayDep() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: "provider.openmcp.cloud", Version: "v1alpha1", Kind: "PlatformService"})
	obj.SetName("gateway")
	return obj
}

func TestGatewayServiceConfig_MutateFn(t *testing.T) {
	dep := testGatewayDep()
	r := GatewayServiceConfig("gateway", dep)

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

func TestGatewayServiceConfig_Dependencies(t *testing.T) {
	dep := testGatewayDep()
	r := GatewayServiceConfig("gateway", dep)
	if len(r.Dependencies) != 1 || r.Dependencies[0] != dep {
		t.Errorf("expected dep as sole dependency, got %v", r.Dependencies)
	}
}

func TestGatewayServiceConfig_GVK(t *testing.T) {
	dep := testGatewayDep()
	r := GatewayServiceConfig("gateway", dep)

	obj := r.Object.(*unstructured.Unstructured)
	gvk := obj.GroupVersionKind()
	if gvk.Group != "gateway.openmcp.cloud" || gvk.Version != "v1alpha1" || gvk.Kind != "GatewayServiceConfig" {
		t.Errorf("unexpected GVK: %v", gvk)
	}
	if obj.GetName() != "gateway" {
		t.Errorf("name = %q, want %q", obj.GetName(), "gateway")
	}
}
