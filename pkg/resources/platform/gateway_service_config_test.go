package platform

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGatewayServiceConfig_MutateFn(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := GatewayServiceConfig("gateway", deployment)

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
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := GatewayServiceConfig("gateway", deployment)
	if len(r.Dependencies) != 1 || r.Dependencies[0] != deployment {
		t.Errorf("expected deployment as sole dependency, got %v", r.Dependencies)
	}
}

func TestGatewayServiceConfig_GVK(t *testing.T) {
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "op", Namespace: "ns"}}
	r := GatewayServiceConfig("gateway", deployment)

	obj := r.Object.(*unstructured.Unstructured)
	gvk := obj.GroupVersionKind()
	if gvk.Group != "gateway.openmcp.cloud" || gvk.Version != "v1alpha1" || gvk.Kind != "GatewayServiceConfig" {
		t.Errorf("unexpected GVK: %v", gvk)
	}
	if obj.GetName() != "gateway" {
		t.Errorf("name = %q, want %q", obj.GetName(), "gateway")
	}
}
