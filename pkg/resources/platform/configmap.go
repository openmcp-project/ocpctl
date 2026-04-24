package platform

import (
	"context"
	"fmt"
	"strings"

	"github.com/ValentinGerlach/oink/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getOperatorConfig(environment string) string {
	return strings.TrimSpace(fmt.Sprintf(`
managedControlPlane:
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
        metadata:
          annotations:
            kind.clusters.openmcp.cloud/name: %s-onboarding
        spec:
          profile: kind
          tenancy: Shared
    workload:
      template:
        spec:
          profile: kind
          tenancy: Shared
`, environment))
}

// OperatorConfigMap returns a Resource for the openmcp-operator ConfigMap
// containing the scheduler and control plane configuration. Depends on the namespace.
func OperatorConfigMap(environment string, ns *corev1.Namespace) *resources.Resource {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "openmcp-operator",
			Namespace: ns.Name,
		},
	}

	return &resources.Resource{
		Object:       cm,
		Dependencies: []client.Object{ns},
		MutateFn: func(ctx context.Context) error {
			cm.Data = map[string]string{
				"config": getOperatorConfig(environment),
			}
			return nil
		},
	}
}
