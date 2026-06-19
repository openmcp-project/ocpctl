package platform

import (
	"context"

	"github.com/openmcp-project/ocpctl/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// the platform-service-gateway pod requires on startup. Depends on the operator
// deployment so it is created before the gateway pod starts.
func GatewayServiceConfig(name string, deployment *appsv1.Deployment) *resources.Resource {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gatewayServiceConfigGVK)
	obj.SetName(name)

	return &resources.Resource{
		Object:       obj,
		Dependencies: []client.Object{deployment},
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
