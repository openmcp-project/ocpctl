// Package platformmesh integrates an OpenControlPlane environment with a
// Platform Mesh KCP instance using separate kind clusters.
package platformmesh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	opcrdfs "github.com/openmcp-project/openmcp-operator/api/crds"
	"github.com/openmcp-project/ocpctl/pkg/logging"
	"k8s.io/client-go/tools/clientcmd"
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
	// create-workspace is a kubectl plugin; global flags must come AFTER the
	// plugin name, so --kubeconfig can't be prepended. Pass via KUBECONFIG env var.
	// We target root:providers explicitly with --server so the workspace type
	// constraint (provider workspaces must live under root:providers) is satisfied
	// regardless of which workspace the kubeconfig's current context points to.
	cmd := exec.CommandContext(ctx, "kubectl",
		"create-workspace", "openmcp-provider",
		"--server="+kcpServer+"/clusters/root:providers",
		"--type=root:provider",
		"--ignore-existing",
	)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kcpKubeconfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

// crdFromEmbeddedFS maps a CRD name to its filename in opcrdfs.CRDFS.
var crdFromEmbeddedFS = map[string]string{
	"managedcontrolplanev2s.core.openmcp.cloud": "manifests/core.openmcp.cloud_managedcontrolplanev2s.yaml",
}

// crdFromOnboarding lists CRDs that are only available in the onboarding cluster
// (not shipped in any Go module dependency).
var crdFromOnboarding = []string{
	"fluxes.flux.services.openmcp.cloud",
}

// step4ApplyCRDs applies the openmcp CRDs to the platform cluster.
// CRDs bundled in the openmcp-operator/api module are applied directly from
// the embedded FS; CRDs only available in the onboarding cluster are copied
// from there.
func step4ApplyCRDs(ctx context.Context, environment, platformCtx string) error {
	// Apply CRDs from the embedded FS in openmcp-operator/api.
	for crdName, path := range crdFromEmbeddedFS {
		data, err := opcrdfs.CRDFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded CRD %s: %w", crdName, err)
		}
		if err := kubectlApplyBytesServerSide(ctx, "", platformCtx, data); err != nil {
			return fmt.Errorf("applying CRD %s: %w", crdName, err)
		}
	}

	// Apply CRDs sourced from the onboarding cluster.
	onboardingCtx := "kind-" + environment + "-onboarding"
	for _, crdName := range crdFromOnboarding {
		yamlBytes, err := kubectlOutput(ctx, "", onboardingCtx,
			"get", "crd", crdName, "-o", "yaml",
		)
		if err != nil {
			return fmt.Errorf("getting CRD %s from %s: %w", crdName, onboardingCtx, err)
		}
		cleaned, err := stripCRDMetadata(yamlBytes)
		if err != nil {
			return fmt.Errorf("stripping metadata from CRD %s: %w", crdName, err)
		}
		if err := kubectlApplyBytesServerSide(ctx, "", platformCtx, cleaned); err != nil {
			return fmt.Errorf("applying CRD %s to %s: %w", crdName, platformCtx, err)
		}
	}
	return nil
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
	rbac := `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: api-syncagent-openmcp
rules:
  - apiGroups: ["core.openmcp.cloud"]
    resources: ["managedcontrolplanev2s", "managedcontrolplanev2s/status"]
    verbs: ["*"]
  - apiGroups: ["flux.services.openmcp.cloud"]
    resources: ["fluxes", "fluxes/status"]
    verbs: ["*"]
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
`
	if err := kubectlApplyContext(ctx, platformCtx, rbac); err != nil {
		return fmt.Errorf("applying RBAC: %w", err)
	}

	publishedResources := `
apiVersion: syncagent.kcp.io/v1alpha1
kind: PublishedResource
metadata:
  name: managed-control-plane
spec:
  resource:
    kind: ManagedControlPlaneV2
    apiGroup: core.openmcp.cloud
    version: v2alpha1
---
apiVersion: syncagent.kcp.io/v1alpha1
kind: PublishedResource
metadata:
  name: flux
spec:
  resource:
    kind: Flux
    apiGroup: flux.services.openmcp.cloud
    version: v1alpha1
`
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
                "url": "https://{context.organization}.portal.localhost:8443/ui/openmcp/#/",
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
// Helpers — thin wrappers around kubectl / helm
// ---------------------------------------------------------------------------

// rewriteKubeconfigServer replaces the KCP server address in a kubeconfig so
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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

// helm runs the helm CLI.
func helm(ctx context.Context, args ...string) error {
	_, err := helmOutput(ctx, args...)
	return err
}

// helmOutput runs helm and returns stdout.
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
