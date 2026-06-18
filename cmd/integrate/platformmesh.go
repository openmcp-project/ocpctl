package integrate

import (
	"fmt"

	integratepkg "github.com/openmcp-project/ocpctl/pkg/integrate/platformmesh"
	"github.com/spf13/cobra"
)

func newPlatformMeshCmd() *cobra.Command {
	var kcpKubeconfig string
	var environment string

	cmd := &cobra.Command{
		Use:   "platform-mesh",
		Short: "Integrate an OpenControlPlane environment with Platform Mesh",
		Long: `Integrates an OpenControlPlane environment with a Platform Mesh KCP instance.

The api-syncagent runs on the kind-<env>-platform cluster; KCP runs on a
separate kind-platform-mesh cluster.

Runs all steps idempotently — already-completed steps are skipped automatically.

Required:
  --kcp-kubeconfig   Path to the KCP admin kubeconfig (e.g. ~/.../kcp/admin.kubeconfig)

Optional:
  --environment        Name of the ocpctl environment (default: "local")

Steps performed:
  Phase A:
    1. Create provider workspace root:providers:openmcp-provider
    2. Create APIExport openmcp.cloud
    3. Grant anonymous bind permissions
    4. Apply openmcp CRDs to platform cluster
    5. Create open-mcp-provider namespace on platform cluster
    6. Rewrite KCP kubeconfig and create secret
    7. Install api-syncagent via Helm
    8. Create syncagent RBAC and PublishedResources
  Phase B:
    9.  Label APIExport and create ProviderMetadata
    10. Create ContentConfiguration in root:platform-mesh-system
  Phase C (OIDC trust ring):
    11. Apply cp-kind ClusterAccess writer RBAC to kind-platform-mesh
    12. Apply GW SA cluster-creator RBAC to kind-<env>-platform
    13. Mount AuthenticationConfiguration on bootstrap clusters`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kcpKubeconfig == "" {
				return fmt.Errorf("--kcp-kubeconfig is required")
			}
			return integratepkg.Run(cmd.Context(), integratepkg.Options{
				KCPKubeconfig: kcpKubeconfig,
				Environment:   environment,
			})
		},
	}

	cmd.Flags().StringVar(&kcpKubeconfig, "kcp-kubeconfig", "", "path to the KCP admin kubeconfig")
	cmd.Flags().StringVar(&environment, "environment", "local", "name of the ocpctl environment")

	return cmd
}
