package platform

import (
	"github.com/ValentinGerlach/oink/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const operatorConfig = `managedControlPlane:
  mcpClusterPurpose: mcp
scheduler:
  scope: Cluster
  purposeMappings:
    mcp:
      template:
        spec:
          profile: kind
          tenancy: Exclusive
    platform:
      template:
        spec:
          profile: kind
          tenancy: Shared
    onboarding:
      template:
        spec:
          profile: kind
          tenancy: Shared
    workload:
      template:
        spec:
          profile: kind
          tenancy: Shared
`

// OperatorConfigMap returns a Resource for the openmcp-operator ConfigMap
// containing the scheduler and control plane configuration. Depends on the namespace.
func OperatorConfigMap(ns *corev1.Namespace) *resources.Resource {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openmcp-operator",
			Namespace: ns.Name,
		},
		Data: map[string]string{
			"config": operatorConfig,
		},
	}

	return &resources.Resource{
		Object:       cm,
		Dependencies: []client.Object{ns},
	}
}
