// Package platformmesh integrates an OpenControlPlane environment with a
// Platform Mesh KCP instance using separate kind clusters.
package platformmesh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/openmcp-project/ocpctl/pkg/logging"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/yaml"
)

// Options holds the configuration for the integration run.
type Options struct {
	// KCPKubeconfig is the path to the KCP admin kubeconfig (e.g. ~/.secret/kcp/admin.kubeconfig).
	KCPKubeconfig string
	// Environment is the ocpctl environment name (e.g. "local"), which maps to
	// the kind cluster "kind-<environment>-platform".
	Environment string
}

// cpKindClusterAccessWriterRBAC is applied to kind-platform-mesh.
// It grants the cp-kind controller-manager OIDC identity permission to write
// ClusterAccess CRs and their CA Secrets in the graphql-gateway namespace.
const cpKindClusterAccessWriterRBAC = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-provider-kind-writer
rules:
  - apiGroups:
      - gateway.platform-mesh.io
    resources:
      - clusteraccesses
    verbs:
      - create
      - get
      - patch
      - update
      - delete
  - apiGroups:
      - ""
    resources:
      - secrets
    verbs:
      - create
      - get
      - patch
      - update
      - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cluster-provider-kind-writer
  namespace: graphql-gateway
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-provider-kind-writer
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: "system:serviceaccount:cluster-provider-kind:controller-manager"
`

// gwClusterCreatorRBAC is applied to kind-<env>-platform.
// It grants the GraphQL Gateway SA OIDC identity permission to create Cluster CRs
// handled by cluster-provider-kind.
const gwClusterCreatorRBAC = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gql-gateway-cluster-creator
rules:
  - apiGroups:
      - clusters.openmcp.cloud
    resources:
      - clusters
    verbs:
      - create
      - get
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: gql-gateway-cluster-creator
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: gql-gateway-cluster-creator
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: "system:serviceaccount:platform-mesh-system:kubernetes-graphql-gateway"
`

// Run executes all integration steps idempotently.
func Run(ctx context.Context, opts Options) error {
	log := logging.FromContext(ctx)

	platformCtx := "kind-" + opts.Environment + "-platform"

	kcpServer, err := kcpServerFromKubeconfig(opts.KCPKubeconfig)
	if err != nil {
		return fmt.Errorf("reading KCP server from kubeconfig: %w", err)
	}

	log.Infof("Starting Platform Mesh integration")
	log.Infof("  KCP kubeconfig : %s", opts.KCPKubeconfig)
	log.Infof("  KCP server     : %s", kcpServer)
	log.Infof("  Platform cluster: %s", platformCtx)

	steps := []step{
		{name: "Step 1: Create provider workspace", fn: func() error {
			return step1ProviderWorkspace(ctx, opts.KCPKubeconfig, kcpServer)
		}},
		{name: "Step 2: Create APIExport", fn: func() error {
			return step2APIExport(ctx, opts.KCPKubeconfig, kcpServer)
		}},
		{name: "Step 3: Grant bind permissions", fn: func() error {
			return step3BindPermissions(ctx, opts.KCPKubeconfig, kcpServer)
		}},
		{name: "Step 4: Apply openmcp CRDs to platform cluster", fn: func() error {
			return step4ApplyCRDs(ctx, opts.Environment, platformCtx)
		}},
		{name: "Step 5: Create open-mcp-provider namespace", fn: func() error {
			return step5Namespace(ctx, platformCtx)
		}},
		{name: "Step 6: Create KCP kubeconfig secret", fn: func() error {
			return step6KubeconfigSecret(ctx, opts.KCPKubeconfig, platformCtx, kcpServer)
		}},
		{name: "Step 7: Install api-syncagent", fn: func() error {
			return step7SyncAgent(ctx, platformCtx, kcpServer)
		}},
		{name: "Step 8: RBAC and PublishedResources", fn: func() error {
			return step8RBACAndPublishedResources(ctx, platformCtx)
		}},
		{name: "Step 9: Label APIExport and create ProviderMetadata", fn: func() error {
			return step9ProviderMetadata(ctx, opts.KCPKubeconfig, kcpServer)
		}},
		{name: "Step 10: Create ContentConfiguration", fn: func() error {
			return step10ContentConfiguration(ctx, opts.KCPKubeconfig, kcpServer)
		}},
		{name: "Step 11: Apply cp-kind ClusterAccess writer RBAC to kind-platform-mesh", fn: func() error {
			return step11ClusterAccessWriterRBAC(ctx)
		}},
		{name: "Step 12: Apply GW SA cluster-creator RBAC to kind-<env>-platform", fn: func() error {
			return step12GWClusterCreatorRBAC(ctx, platformCtx)
		}},
		{name: "Step 13: Create cp-kind credential on kind-platform-mesh", fn: func() error {
			return step13CpKindCredential(ctx, platformCtx)
		}},
		{name: "Step 14: Write platform-mesh CA Secret to kind-platform-mesh", fn: func() error {
			return step14PlatformMeshCASecret(ctx)
		}},
		{name: "Step 15: Write local-platform ClusterAccess to kind-platform-mesh", fn: func() error {
			return step15LocalPlatformClusterAccess(ctx, opts.Environment)
		}},
	}

	for _, s := range steps {
		log.Infof("→ %s", s.name)
		if err := s.fn(); err != nil {
			return fmt.Errorf("%s: %w", s.name, err)
		}
		log.Infof("  ✓ done")
	}

	log.Info("Platform Mesh integration complete.")
	return nil
}

type step struct {
	name string
	fn   func() error
}

// ---------------------------------------------------------------------------
// Step 1 — Create provider workspace
// ---------------------------------------------------------------------------

func step1ProviderWorkspace(ctx context.Context, kcpKubeconfig, kcpServer string) error {
	// create-workspace is a kubectl plugin that does not accept --kubeconfig as a
	// flag; credentials must come via the KUBECONFIG env var.
	//
	// Passing --server alongside KUBECONFIG causes kubectl to use the given server
	// URL but drop the credentials from the kubeconfig context, resulting in an
	// unauthenticated request (User ""). Instead, navigate using `kubectl kcp
	// workspace use` so that create-workspace inherits both the server and the
	// credentials from the kubeconfig context.
	//
	// The `provider` workspace type requires its parent to be of type `providers`
	// (limitAllowedParents), so we ensure root:providers exists first.
	env := append(os.Environ(), "KUBECONFIG="+kcpKubeconfig)

	// Step 1a: navigate to root and create the `providers` parent workspace.
	for _, useArg := range []string{":root"} {
		use := exec.CommandContext(ctx, "kubectl", "kcp", "workspace", "use", useArg)
		use.Env = env
		use.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
		use.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
		if err := use.Run(); err != nil {
			return fmt.Errorf("switching to %s workspace: %w", useArg, err)
		}
	}

	createProviders := exec.CommandContext(ctx, "kubectl",
		"create-workspace", "providers",
		"--type=root:providers",
		"--ignore-existing",
	)
	createProviders.Env = env
	createProviders.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	createProviders.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	if err := createProviders.Run(); err != nil {
		return fmt.Errorf("creating providers workspace: %w", err)
	}

	// Step 1b: navigate into root:providers and create the openmcp-provider workspace.
	use := exec.CommandContext(ctx, "kubectl", "kcp", "workspace", "use", ":root:providers")
	use.Env = env
	use.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	use.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	if err := use.Run(); err != nil {
		return fmt.Errorf("switching to root:providers workspace: %w", err)
	}

	create := exec.CommandContext(ctx, "kubectl",
		"create-workspace", "openmcp-provider",
		"--type=root:provider",
		"--ignore-existing",
	)
	create.Env = env
	create.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	create.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	return create.Run()
}

// ---------------------------------------------------------------------------
// Step 2 — Create APIExport
// ---------------------------------------------------------------------------

func step2APIExport(ctx context.Context, kcpKubeconfig, kcpServer string) error {
	providerServer := kcpServer + "/clusters/root:providers:openmcp-provider"
	manifest := `
apiVersion: apis.kcp.io/v1alpha1
kind: APIExport
metadata:
  name: openmcp.cloud
spec: {}
`
	return kubectlApply(ctx, kcpKubeconfig, providerServer, manifest)
}

// ---------------------------------------------------------------------------
// Step 3 — Grant bind permissions
// ---------------------------------------------------------------------------

func step3BindPermissions(ctx context.Context, kcpKubeconfig, kcpServer string) error {
	providerServer := kcpServer + "/clusters/root:providers:openmcp-provider"
	manifest := `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apiexport-bind
rules:
  - apiGroups: ["apis.kcp.io"]
    resources: ["apiexports"]
    verbs: ["bind"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: anonymous-view
subjects:
  - kind: User
    name: system:anonymous
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: apiexport-bind
  apiGroup: rbac.authorization.k8s.io
`
	return kubectlApply(ctx, kcpKubeconfig, providerServer, manifest)
}

// ---------------------------------------------------------------------------
// Step 4 — Apply openmcp CRDs to platform cluster
// ---------------------------------------------------------------------------

// crdFromOnboarding lists CRDs to copy from the onboarding cluster to the platform cluster.
// These are installed on the onboarding cluster by their respective operators at startup.
// The bool indicates whether the CRD is required; optional CRDs are skipped when absent.
var crdFromOnboarding = []struct {
	name     string
	required bool
}{
	{"controlplanes.core.open-control-plane.io", true},
	{"fluxes.flux.services.open-control-plane.io", false},
}

// step4ApplyCRDs applies the openmcp CRDs to the platform cluster by copying
// them from the onboarding cluster where the operators have already installed them.
// Optional CRDs are skipped when not present on the onboarding cluster.
func step4ApplyCRDs(ctx context.Context, environment, platformCtx string) error {
	log := logging.FromContext(ctx)
	onboardingCtx := "kind-" + environment + "-onboarding"
	for _, crd := range crdFromOnboarding {
		yamlBytes, err := kubectlOutput(ctx, "", onboardingCtx,
			"get", "crd", crd.name, "-o", "yaml",
		)
		if err != nil {
			if !crd.required && isNotFound(err) {
				log.Infof("  optional CRD %s not found on %s, skipping", crd.name, onboardingCtx)
				continue
			}
			return fmt.Errorf("getting CRD %s from %s: %w", crd.name, onboardingCtx, err)
		}
		cleaned, err := stripCRDMetadata(yamlBytes)
		if err != nil {
			return fmt.Errorf("stripping metadata from CRD %s: %w", crd.name, err)
		}
		if err := kubectlApplyBytesServerSide(ctx, "", platformCtx, cleaned); err != nil {
			return fmt.Errorf("applying CRD %s to %s: %w", crd.name, platformCtx, err)
		}
	}
	return nil
}

// isNotFound reports whether an error from kubectlOutput is a Kubernetes NotFound error.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NotFound")
}

// stripCRDMetadata removes resourceVersion, uid, generation, and status from
// a CRD YAML blob so it can be safely applied to a different cluster.
func stripCRDMetadata(data []byte) ([]byte, error) {
	var obj map[string]interface{}
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if meta, ok := obj["metadata"].(map[string]interface{}); ok {
		delete(meta, "resourceVersion")
		delete(meta, "uid")
		delete(meta, "generation")
		delete(meta, "creationTimestamp")
		delete(meta, "managedFields")
		delete(meta, "annotations") // may contain last-applied-configuration from source cluster
	}
	delete(obj, "status")
	return yaml.Marshal(obj)
}

// ---------------------------------------------------------------------------
// Step 5 — Create namespace
// ---------------------------------------------------------------------------

const providerNamespace = "open-mcp-provider"

func step5Namespace(ctx context.Context, platformCtx string) error {
	// create namespace is idempotent via --dry-run=client | apply
	manifest := fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, providerNamespace)
	return kubectlApplyContext(ctx, platformCtx, manifest)
}

// ---------------------------------------------------------------------------
// Step 6 — Create KCP kubeconfig secret (Option B)
// ---------------------------------------------------------------------------

func step6KubeconfigSecret(ctx context.Context, kcpKubeconfig, platformCtx, kcpServer string) error {
	log := logging.FromContext(ctx)

	// Check if secret already exists — skip rewriting and re-creating if so.
	out, err := kubectlOutput(ctx, "", platformCtx,
		"get", "secret", "open-mcp-kubeconfig",
		"-n", providerNamespace,
		"--ignore-not-found",
		"-o", "name",
	)
	if err != nil {
		return fmt.Errorf("checking for existing secret: %w", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		log.Info("  secret open-mcp-kubeconfig already exists, skipping")
		return nil
	}

	// Get Traefik node IP and nodePort from the platform-mesh kind cluster.
	traefikIP, err := kubectlOutput(ctx, "", "kind-platform-mesh",
		"get", "nodes",
		"-o", `jsonpath={.items[0].status.addresses[?(@.type=="InternalIP")].address}`,
	)
	if err != nil {
		return fmt.Errorf("getting traefik node IP: %w", err)
	}
	traefikPort, err := kubectlOutput(ctx, "", "kind-platform-mesh",
		"get", "svc", "traefik", "-n", "default",
		"-o", `jsonpath={.spec.ports[?(@.port==8443)].nodePort}`,
	)
	if err != nil {
		return fmt.Errorf("getting traefik nodePort: %w", err)
	}

	ip := strings.TrimSpace(string(traefikIP))
	port := strings.TrimSpace(string(traefikPort))
	if ip == "" {
		return fmt.Errorf("could not determine Traefik node IP from kind-platform-mesh")
	}
	if port == "" {
		return fmt.Errorf("could not determine Traefik nodePort from kind-platform-mesh")
	}
	log.Infof("  Traefik: %s:%s", ip, port)

	// Rewrite the kubeconfig: replace localhost:8443 with localhost:<nodePort>.
	// The syncagent pod uses hostAliases to resolve localhost → Traefik IP.
	raw, err := os.ReadFile(kcpKubeconfig)
	if err != nil {
		return fmt.Errorf("reading kcp kubeconfig: %w", err)
	}
	rewritten := rewriteKubeconfigServer(string(raw), kcpServer, port)

	tmp, err := os.CreateTemp("", "kcp-kubeconfig-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(rewritten); err != nil {
		return fmt.Errorf("writing temp kubeconfig: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp kubeconfig: %w", err)
	}

	return kubectlRaw(ctx, "", platformCtx,
		"create", "secret", "generic", "open-mcp-kubeconfig",
		"-n", providerNamespace,
		"--from-file=kubeconfig="+tmp.Name(),
	)
}

// ---------------------------------------------------------------------------
// Step 7 — Install api-syncagent via Helm
// ---------------------------------------------------------------------------

func step7SyncAgent(ctx context.Context, platformCtx, kcpServer string) error {
	log := logging.FromContext(ctx)

	// Check if already installed.
	out, err := helmOutput(ctx, "list",
		"-n", providerNamespace,
		"--kube-context", platformCtx,
		"-o", "json",
	)
	if err != nil {
		return fmt.Errorf("listing helm releases: %w", err)
	}

	var releases []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(out, &releases); err != nil {
		return fmt.Errorf("parsing helm list output: %w", err)
	}
	for _, r := range releases {
		if r.Name == "api-syncagent" {
			log.Info("  api-syncagent Helm release already installed, skipping")
			return nil
		}
	}

	// Get Traefik IP for hostAliases.
	traefikIP, err := kubectlOutput(ctx, "", "kind-platform-mesh",
		"get", "nodes",
		"-o", `jsonpath={.items[0].status.addresses[?(@.type=="InternalIP")].address}`,
	)
	if err != nil {
		return fmt.Errorf("getting traefik IP: %w", err)
	}
	ip := strings.TrimSpace(string(traefikIP))
	if ip == "" {
		return fmt.Errorf("could not determine Traefik IP")
	}

	// Add/update the kcp helm repo.
	if err := helm(ctx, "repo", "add", "kcp", "https://kcp-dev.github.io/helm-charts"); err != nil {
		// "already exists" is not fatal
		log.Debugf("helm repo add (may already exist): %v", err)
	}
	if err := helm(ctx, "repo", "update"); err != nil {
		return fmt.Errorf("helm repo update: %w", err)
	}

	return helm(ctx,
		"install", "api-syncagent", "kcp/api-syncagent",
		"--kube-context", platformCtx,
		"-n", providerNamespace,
		"--set", "replicaCount=1",
		"--set", "apiExportEndpointSliceName=openmcp.cloud",
		"--set", "agentName=open-mcp-syncagent",
		"--set", "kcpKubeconfig=open-mcp-kubeconfig",
		"--set", "hostAliases.enabled=true",
		"--set", "hostAliases.values[0].ip="+ip,
		"--set", "hostAliases.values[0].hostnames[0]=localhost",
	)
}

// ---------------------------------------------------------------------------
// Step 8 — RBAC and PublishedResources
// ---------------------------------------------------------------------------

func step8RBACAndPublishedResources(ctx context.Context, platformCtx string) error {
	log := logging.FromContext(ctx)

	// Determine whether the Flux CRD landed on the platform cluster.
	_, fluxErr := kubectlOutput(ctx, "", platformCtx,
		"get", "crd", "fluxes.flux.services.open-control-plane.io", "--ignore-not-found", "-o", "name",
	)
	fluxAvailable := fluxErr == nil

	fluxRBACRule := ""
	if fluxAvailable {
		fluxRBACRule = `
  - apiGroups: ["flux.services.open-control-plane.io"]
    resources: ["fluxes", "fluxes/status"]
    verbs: ["*"]`
	} else {
		log.Info("  Flux CRD not present on platform cluster, skipping Flux RBAC and PublishedResource")
	}

	rbac := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: api-syncagent-openmcp
rules:
  - apiGroups: ["core.orchestrate.cloud.sap"]
    resources: ["controlplanes", "controlplanes/status"]
    verbs: ["*"]%s
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: api-syncagent-openmcp
subjects:
  - kind: ServiceAccount
    name: api-syncagent
    namespace: open-mcp-provider
roleRef:
  kind: ClusterRole
  name: api-syncagent-openmcp
  apiGroup: rbac.authorization.k8s.io
`, fluxRBACRule)
	if err := kubectlApplyContext(ctx, platformCtx, rbac); err != nil {
		return fmt.Errorf("applying RBAC: %w", err)
	}

	publishedResources := `
apiVersion: syncagent.kcp.io/v1alpha1
kind: PublishedResource
metadata:
  name: control-plane
spec:
  resource:
    kind: ControlPlane
    apiGroup: core.orchestrate.cloud.sap
    version: v1beta1
`
	if fluxAvailable {
		publishedResources += `---
apiVersion: syncagent.kcp.io/v1alpha1
kind: PublishedResource
metadata:
  name: flux
spec:
  resource:
    kind: Flux
    apiGroup: flux.services.open-control-plane.io
    version: v1alpha1
`
	}
	if err := kubectlApplyContext(ctx, platformCtx, publishedResources); err != nil {
		return fmt.Errorf("applying PublishedResources: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Step 9 — Label APIExport and create ProviderMetadata
// ---------------------------------------------------------------------------

func step9ProviderMetadata(ctx context.Context, kcpKubeconfig, kcpServer string) error {
	providerServer := kcpServer + "/clusters/root:providers:openmcp-provider"

	// Label the APIExport (idempotent — label apply is safe to re-run).
	if err := kubectlRaw(ctx, kcpKubeconfig, "",
		"--server="+providerServer,
		"label", "apiexport", "openmcp.cloud",
		"ui.platform-mesh.io/content-for=openmcp.cloud",
		"--overwrite",
	); err != nil {
		return fmt.Errorf("labelling APIExport: %w", err)
	}

	manifest := `
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ProviderMetadata
metadata:
  name: openmcp.cloud
spec:
  displayName: OpenMCP
  description: >
    OpenMCP provides managed control planes and Flux-based GitOps services
    for Kubernetes clusters.
  tags:
    - control-plane
    - gitops
    - flux
  icon:
    light:
      url: https://github.com/openmcp-project/.github/raw/main/profile/opencontrolplane_logo.svg
    dark:
      url: https://github.com/openmcp-project/.github/raw/main/profile/opencontrolplane_logo.svg
  contacts:
    - displayName: OpenMCP Team
      email: ManagedControlPlanes@sap.com
      role:
        - owner
  documentation:
    - displayName: OpenMCP Docs
      url: https://open-control-plane.io/operators/quickstart
  preferredSupportChannels:
    - displayName: GitHub Issues
      url: https://github.com/openmcp-project/backlog/issues
`
	return kubectlApply(ctx, kcpKubeconfig, providerServer, manifest)
}

// ---------------------------------------------------------------------------
// Step 10 — Create ContentConfiguration
// ---------------------------------------------------------------------------

func step10ContentConfiguration(ctx context.Context, kcpKubeconfig, kcpServer string) error {
	systemServer := kcpServer + "/clusters/root:platform-mesh-system"

	manifest := `
apiVersion: ui.platform-mesh.io/v1alpha1
kind: ContentConfiguration
metadata:
  name: account-openmcp
  labels:
    ui.platform-mesh.io/entity: core_platform-mesh_io_account
spec:
  inlineConfiguration:
    contentType: json
    content: |
      {
        "name": "openmcp",
        "luigiConfigFragment": {
          "data": {
            "nodes": [
              {
                "pathSegment": "openmcp",
                "entityType": "main.core_platform-mesh_io_account",
                "order": 800,
                "label": "OpenMCP",
                "icon": "product",
                "url": "http://{context.organization}.portal.localhost:4200/",
                "context": {
                  "accountId": ":core_platform-mesh_io_accountId"
                }
              }
            ],
            "texts": [
              {
                "locale": "en",
                "textDictionary": {
                  "openmcp": "OpenMCP"
                }
              }
            ]
          }
        }
      }
`
	return kubectlApply(ctx, kcpKubeconfig, systemServer, manifest)
}

// ---------------------------------------------------------------------------
// Steps 11-13 — OIDC trust ring
// ---------------------------------------------------------------------------

// step11ClusterAccessWriterRBAC applies the cp-kind ClusterAccess writer RBAC
// to kind-platform-mesh.
func step11ClusterAccessWriterRBAC(ctx context.Context) error {
	return kubectlApplyContext(ctx, "kind-platform-mesh", cpKindClusterAccessWriterRBAC)
}

// step12GWClusterCreatorRBAC applies the GW SA cluster-creator RBAC to
// kind-<env>-platform.
func step12GWClusterCreatorRBAC(ctx context.Context, platformCtx string) error {
	return kubectlApplyContext(ctx, platformCtx, gwClusterCreatorRBAC)
}

// step13CpKindCredential creates a static ServiceAccount + token Secret on
// kind-platform-mesh for the cp-kind controller-manager, then writes a kubeconfig
// Secret onto kind-<env>-platform so cp-kind can mount and use it.
//
// The token is of type kubernetes.io/service-account-token (non-expiring, signed
// by kind-platform-mesh's own SA key). No signing key crosses cluster boundaries.
func step13CpKindCredential(ctx context.Context, platformCtx string) error {
	log := logging.FromContext(ctx)
	const (
		pmContext    = "kind-platform-mesh"
		saNamespace  = "cluster-provider-kind"
		saName       = "controller-manager"
		secretName   = "controller-manager-token"
		kcfgSecret   = "platform-mesh-kubeconfig"
	)

	// Ensure the namespace exists.
	nsManifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, saNamespace)
	if err := kubectlApplyContext(ctx, pmContext, nsManifest); err != nil {
		return fmt.Errorf("ensuring namespace %s on %s: %w", saNamespace, pmContext, err)
	}

	// Create the ServiceAccount.
	saManifest := fmt.Sprintf(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
`, saName, saNamespace)
	if err := kubectlApplyContext(ctx, pmContext, saManifest); err != nil {
		return fmt.Errorf("applying ServiceAccount: %w", err)
	}

	// Create the static token Secret (type kubernetes.io/service-account-token).
	// The kube controller manager populates .data.token and .data.ca.crt automatically.
	tokenSecretManifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  annotations:
    kubernetes.io/service-account.name: %s
type: kubernetes.io/service-account-token
`, secretName, saNamespace, saName)
	if err := kubectlApplyContext(ctx, pmContext, tokenSecretManifest); err != nil {
		return fmt.Errorf("applying token Secret: %w", err)
	}

	// Wait for the controller manager to populate the token (max 30s).
	log.Infof("  waiting for token to be populated in %s/%s", saNamespace, secretName)
	var token string
	deadline := time.Now().Add(30 * time.Second)
	for {
		t, err := kubectlOutput(ctx, "", pmContext,
			"get", "secret", secretName, "-n", saNamespace,
			"-o", "jsonpath={.data.token}",
		)
		if err == nil && len(strings.TrimSpace(string(t))) > 0 {
			token = strings.TrimSpace(string(t))
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("token Secret %s/%s was not populated within 30s", saNamespace, secretName)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	// Get the kind-platform-mesh API server address via the Docker-internal kubeconfig
	// (internal=true) so the address is reachable from within the
	// kind-local-platform pod network.
	provider := cluster.NewProvider()
	pmKubeconfigRaw, err := provider.KubeConfig("platform-mesh", true)
	if err != nil {
		return fmt.Errorf("getting internal kubeconfig for kind-platform-mesh: %w", err)
	}
	pmCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(pmKubeconfigRaw))
	if err != nil {
		return fmt.Errorf("parsing internal kubeconfig for kind-platform-mesh: %w", err)
	}
	server := pmCfg.Host

	// Build a kubeconfig using the static token.
	// caData is already base64-encoded (from jsonpath .data.*) — correct for
	// certificate-authority-data in a kubeconfig.
	// token is base64-encoded from .data.token — decode it to a plain bearer token.
	tokenDecoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return fmt.Errorf("decoding token: %w", err)
	}
	// Use CA from the kind provider's REST config (already a PEM []byte).
	caPEM := base64.StdEncoding.EncodeToString(pmCfg.TLSClientConfig.CAData)

	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: %s
    server: %s
  name: platform-mesh
contexts:
- context:
    cluster: platform-mesh
    user: cp-kind
  name: platform-mesh
current-context: platform-mesh
users:
- name: cp-kind
  user:
    token: %s
`, caPEM, server, string(tokenDecoded))

	// Base64-encode the kubeconfig for the Secret data field.
	kcfgB64 := base64.StdEncoding.EncodeToString([]byte(kubeconfig))

	// Write the kubeconfig as a Secret onto kind-<env>-platform so cp-kind can mount it.
	kcfgSecretManifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
data:
  kubeconfig: %s
`, kcfgSecret, saNamespace, kcfgB64)

	// Ensure namespace exists on kind-<env>-platform too.
	if err := kubectlApplyContext(ctx, platformCtx, nsManifest); err != nil {
		return fmt.Errorf("ensuring namespace %s on %s: %w", saNamespace, platformCtx, err)
	}
	if err := kubectlApplyContext(ctx, platformCtx, kcfgSecretManifest); err != nil {
		return fmt.Errorf("applying kubeconfig Secret to %s: %w", platformCtx, err)
	}

	log.Infof("  cp-kind kubeconfig Secret %s/%s written to %s", saNamespace, kcfgSecret, platformCtx)
	return nil
}

// ---------------------------------------------------------------------------
// Step 14 — Write platform-mesh CA Secret to kind-platform-mesh
// ---------------------------------------------------------------------------

// step14PlatformMeshCASecret extracts the CA certificate from the kind-platform-mesh
// kubeconfig and writes it as a Secret to kind-platform-mesh so that cp-kind can mount
// it and verify TLS when calling the platform-mesh API server.
//
// Secret: namespace=cluster-provider-kind, name=platform-mesh-ca, key=ca.crt (PEM).
// The operation is idempotent (kubectl apply).
func step14PlatformMeshCASecret(ctx context.Context) error {
	const (
		kindClusterName = "platform-mesh"
		secretNamespace = "cluster-provider-kind"
		secretName      = "platform-mesh-ca"
		contextName     = "kind-platform-mesh"
	)

	provider := cluster.NewProvider()
	kubeconfigRaw, err := provider.KubeConfig(kindClusterName, false)
	if err != nil {
		return fmt.Errorf("getting kubeconfig for kind-%s: %w", kindClusterName, err)
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigRaw))
	if err != nil {
		return fmt.Errorf("parsing kubeconfig for kind-%s: %w", kindClusterName, err)
	}

	caData := cfg.TLSClientConfig.CAData
	if len(caData) == 0 {
		return fmt.Errorf("kind-%s kubeconfig has no CA data", kindClusterName)
	}

	// Ensure the namespace exists before writing the Secret.
	nsManifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, secretNamespace)
	if err := kubectlApplyContext(ctx, contextName, nsManifest); err != nil {
		return fmt.Errorf("ensuring namespace %s: %w", secretNamespace, err)
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
data:
  ca.crt: %s
`, secretName, secretNamespace, base64.StdEncoding.EncodeToString(caData))

	return kubectlApplyContext(ctx, contextName, manifest)
}

// ---------------------------------------------------------------------------
// Step 15 — Write local-platform ClusterAccess to kind-platform-mesh
// ---------------------------------------------------------------------------

// step15LocalPlatformClusterAccess writes a ClusterAccess + CA Secret on
// kind-platform-mesh pointing at kind-<environment>-platform, so the
// GraphQL Gateway can discover and query the local-platform cluster.
//
// cp-kind's reconcileClusterAccess handles this automatically for control-plane
// clusters it creates, but kind-<env>-platform is created by ocpctl (not by
// cp-kind), so cp-kind never runs reconcileClusterAccess for it.
//
// The ClusterAccess uses a ServiceAccountRef so the gateway generates a
// short-lived OIDC token instead of a static credential.
func step15LocalPlatformClusterAccess(ctx context.Context, environment string) error {
	const (
		pmContext        = "kind-platform-mesh"
		caNamespace      = "graphql-gateway"
		gwNamespace      = "platform-mesh-system"
		gwServiceAccount = "kubernetes-graphql-gateway"
	)

	clusterName := environment + "-platform" // e.g. "local-platform"
	kindClusterName := environment + "-platform"

	provider := cluster.NewProvider()
	// Use internal IP (runsOnLocalHost=false) so the host stored in ClusterAccess
	// is reachable from within the kind-platform-mesh Docker network.
	kubeconfigRaw, err := provider.KubeConfig(kindClusterName, false)
	if err != nil {
		return fmt.Errorf("getting kubeconfig for kind-%s: %w", kindClusterName, err)
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigRaw))
	if err != nil {
		return fmt.Errorf("parsing kubeconfig for kind-%s: %w", kindClusterName, err)
	}

	caData := cfg.TLSClientConfig.CAData
	if len(caData) == 0 {
		return fmt.Errorf("kind-%s kubeconfig has no CA data", kindClusterName)
	}

	caSecretName := clusterName + "-ca"

	nsManifest := fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
`, caNamespace)
	if err := kubectlApplyContext(ctx, pmContext, nsManifest); err != nil {
		return fmt.Errorf("ensuring namespace %s: %w", caNamespace, err)
	}

	caManifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
data:
  ca.crt: %s
`, caSecretName, caNamespace, base64.StdEncoding.EncodeToString(caData))

	if err := kubectlApplyContext(ctx, pmContext, caManifest); err != nil {
		return fmt.Errorf("applying CA secret for %s: %w", clusterName, err)
	}

	clusterAccessManifest := fmt.Sprintf(`apiVersion: gateway.platform-mesh.io/v1alpha1
kind: ClusterAccess
metadata:
  name: %s
spec:
  host: %s
  ca:
    secretRef:
      name: %s
      namespace: %s
      key: ca.crt
  auth:
    serviceAccountRef:
      name: %s
      namespace: %s
      audience:
        - kubernetes
      token_expiration: 1h
`, clusterName, cfg.Host, caSecretName, caNamespace, gwServiceAccount, gwNamespace)

	if err := kubectlApplyContext(ctx, pmContext, clusterAccessManifest); err != nil {
		return fmt.Errorf("applying ClusterAccess for %s: %w", clusterName, err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers — thin wrappers around kubectl / helm
// --------------------------------------------------------------------------- replaces the KCP server address in a kubeconfig so
// the syncagent pod can reach KCP via the Traefik nodePort instead of the
// local port-forward address.
func rewriteKubeconfigServer(kubeconfig, oldServer, nodePort string) string {
	u, err := url.Parse(oldServer)
	if err != nil {
		u = &url.URL{Scheme: "https", Host: "localhost:8443"}
	}
	u.Host = fmt.Sprintf("%s:%s", hostWithoutPort(u.Host), nodePort)
	return strings.ReplaceAll(kubeconfig, oldServer, u.String())
}

// hostWithoutPort strips the port from a host:port string.
func hostWithoutPort(hostport string) string {
	if idx := strings.LastIndex(hostport, ":"); idx != -1 {
		return hostport[:idx]
	}
	return hostport
}

// kcpServerFromKubeconfig reads the server URL for the current context's
// cluster from a kubeconfig file.
func kcpServerFromKubeconfig(kubeconfigPath string) (string, error) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return "", fmt.Errorf("loading kubeconfig: %w", err)
	}
	currentContext := cfg.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("kubeconfig has no current-context set")
	}
	ctx, ok := cfg.Contexts[currentContext]
	if !ok {
		return "", fmt.Errorf("context %q not found in kubeconfig", currentContext)
	}
	cluster, ok := cfg.Clusters[ctx.Cluster]
	if !ok {
		return "", fmt.Errorf("cluster %q not found in kubeconfig", ctx.Cluster)
	}
	if cluster.Server == "" {
		return "", fmt.Errorf("cluster %q has no server URL", ctx.Cluster)
	}
	// Strip any path component — we want just scheme+host as the base URL.
	u, err := url.Parse(cluster.Server)
	if err != nil {
		return "", fmt.Errorf("parsing server URL %q: %w", cluster.Server, err)
	}
	u.Path = ""
	return u.String(), nil
}

// indentWriter wraps an io.Writer and prefixes every line with a fixed indent.
type indentWriter struct {
	w      io.Writer
	prefix []byte
	// pending tracks whether the next write starts a new line that needs indenting.
	pending bool
}

func newIndentWriter(w io.Writer, prefix string) *indentWriter {
	return &indentWriter{w: w, prefix: []byte(prefix), pending: true}
}

func (iw *indentWriter) Write(p []byte) (int, error) {
	total := 0
	for len(p) > 0 {
		if iw.pending {
			if _, err := iw.w.Write(iw.prefix); err != nil {
				return total, err
			}
			iw.pending = false
		}
		idx := bytes.IndexByte(p, '\n')
		if idx == -1 {
			n, err := iw.w.Write(p)
			total += n
			return total, err
		}
		n, err := iw.w.Write(p[:idx+1])
		total += n
		if err != nil {
			return total, err
		}
		iw.pending = true
		p = p[idx+1:]
	}
	return total, nil
}

const subprocessIndent = "      "

// kubectlOutput runs kubectl and returns stdout.
func kubectlOutput(ctx context.Context, kubeconfig, contextOrServer string, args ...string) ([]byte, error) {
	full := buildKubectlArgs(kubeconfig, contextOrServer, args)
	cmd := exec.CommandContext(ctx, "kubectl", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("kubectl %s: %w\n%s", strings.Join(full, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// kubectlRaw runs kubectl without capturing output — prints to the caller's stdout/stderr.
func kubectlRaw(ctx context.Context, kubeconfig, contextOrServer string, args ...string) error {
	full := buildKubectlArgs(kubeconfig, contextOrServer, args)
	cmd := exec.CommandContext(ctx, "kubectl", full...)
	cmd.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	cmd.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	return cmd.Run()
}

// kubectlApply applies a YAML manifest string using the given kubeconfig and server URL.
func kubectlApply(ctx context.Context, kubeconfig, server, manifest string) error {
	return kubectlApplyBytes(ctx, kubeconfig, server, []byte(manifest))
}

// kubectlApplyContext applies a manifest using a --context flag instead of a server URL.
func kubectlApplyContext(ctx context.Context, contextName, manifest string) error {
	return kubectlApplyBytes(ctx, "", contextName, []byte(manifest))
}

// kubectlApplyBytesServerSide pipes raw YAML into kubectl apply --server-side -f -.
// Server-side apply does not require resourceVersion, making it safe for cross-cluster copies.
func kubectlApplyBytesServerSide(ctx context.Context, kubeconfig, contextOrServer string, data []byte) error {
	full := buildKubectlArgs(kubeconfig, contextOrServer, []string{"apply", "--server-side", "--force-conflicts", "-f", "-"})
	cmd := exec.CommandContext(ctx, "kubectl", full...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	cmd.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply --server-side: %w", err)
	}
	return nil
}

// kubectlApplyBytes pipes raw YAML into kubectl apply -f -.
func kubectlApplyBytes(ctx context.Context, kubeconfig, contextOrServer string, data []byte) error {
	full := buildKubectlArgs(kubeconfig, contextOrServer, []string{"apply", "-f", "-"})
	cmd := exec.CommandContext(ctx, "kubectl", full...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	cmd.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}
	return nil
}

// buildKubectlArgs prepends --kubeconfig and --context/--server flags as needed.
// contextOrServer is treated as a --server URL when it starts with "https://",
// otherwise as a --context value.
func buildKubectlArgs(kubeconfig, contextOrServer string, args []string) []string {
	var prefix []string
	if kubeconfig != "" {
		prefix = append(prefix, "--kubeconfig="+kubeconfig)
	}
	if contextOrServer != "" {
		if strings.HasPrefix(contextOrServer, "https://") {
			prefix = append(prefix, "--server="+contextOrServer)
		} else {
			prefix = append(prefix, "--context="+contextOrServer)
		}
	}
	return append(prefix, args...)
}

// helm runs the helm CLI, printing output with the subprocess indent.
func helm(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = newIndentWriter(os.Stdout, subprocessIndent)
	cmd.Stderr = newIndentWriter(os.Stderr, subprocessIndent)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// helmOutput runs helm and returns stdout (no indented printing).
func helmOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("helm %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}
