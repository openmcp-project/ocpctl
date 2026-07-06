package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// defaultEnvoyChartTag is the Envoy Gateway Helm chart version.
	// Update when upgrading platform-service-gateway.
	defaultEnvoyChartTag = "1.5.4"
	defaultBaseDomain    = "openmcp.localhost"
)

var gatewayServiceConfigGVK = schema.GroupVersionKind{
	Group:   "gateway.openmcp.cloud",
	Version: "v1alpha1",
	Kind:    "GatewayServiceConfig",
}

// GatewayServiceConfig returns a Resource for the GatewayServiceConfig CR that
// the platform-service-gateway pod requires on startup. The CRD is installed by
// the gateway pod itself; IsNoMatchError retries handle the race automatically.
func GatewayServiceConfig(name string) *resources.Resource {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gatewayServiceConfigGVK)
	obj.SetName(name)

	return &resources.Resource{
		Object:  obj,
		MutateFn: func(_ context.Context) error {
			return unstructured.SetNestedField(obj.Object, map[string]interface{}{
				"dns": map[string]interface{}{
					"baseDomain": defaultBaseDomain,
				},
				"clusters": []interface{}{
					map[string]interface{}{
						"selector": map[string]interface{}{
							"matchPurpose": "platform",
						},
					},
					map[string]interface{}{
						"selector": map[string]interface{}{
							"matchPurpose": "workload",
						},
					},
				},
				"envoyGateway": map[string]interface{}{
					"chart": map[string]interface{}{
						"tag": defaultEnvoyChartTag,
					},
				},
			}, "spec")
		},
	}
}
