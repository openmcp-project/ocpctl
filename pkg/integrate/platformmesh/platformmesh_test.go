package platformmesh

import (
	"os"
	"strings"
	"testing"
)

func TestBuildKubectlArgs(t *testing.T) {
	tests := []struct {
		name            string
		kubeconfig      string
		contextOrServer string
		args            []string
		want            []string
	}{
		{
			name:            "no kubeconfig no context",
			kubeconfig:      "",
			contextOrServer: "",
			args:            []string{"get", "pods"},
			want:            []string{"get", "pods"},
		},
		{
			name:            "kubeconfig only",
			kubeconfig:      "/path/to/kubeconfig",
			contextOrServer: "",
			args:            []string{"get", "pods"},
			want:            []string{"--kubeconfig=/path/to/kubeconfig", "get", "pods"},
		},
		{
			name:            "context name",
			kubeconfig:      "",
			contextOrServer: "kind-local-platform",
			args:            []string{"get", "nodes"},
			want:            []string{"--context=kind-local-platform", "get", "nodes"},
		},
		{
			name:            "server URL routed as --server",
			kubeconfig:      "",
			contextOrServer: "https://localhost:8443/clusters/root:providers:openmcp-provider",
			args:            []string{"apply", "-f", "-"},
			want:            []string{"--server=https://localhost:8443/clusters/root:providers:openmcp-provider", "apply", "-f", "-"},
		},
		{
			name:            "kubeconfig and server URL",
			kubeconfig:      "/kcp/admin.kubeconfig",
			contextOrServer: "https://localhost:8443/clusters/root:providers",
			args:            []string{"get", "workspaces"},
			want:            []string{"--kubeconfig=/kcp/admin.kubeconfig", "--server=https://localhost:8443/clusters/root:providers", "get", "workspaces"},
		},
		{
			name:            "kubeconfig and context",
			kubeconfig:      "/kcp/admin.kubeconfig",
			contextOrServer: "kind-local-platform",
			args:            []string{"get", "pods"},
			want:            []string{"--kubeconfig=/kcp/admin.kubeconfig", "--context=kind-local-platform", "get", "pods"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKubectlArgs(tt.kubeconfig, tt.contextOrServer, tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("buildKubectlArgs() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStripCRDMetadata(t *testing.T) {
	input := `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: fluxes.flux.services.openmcp.cloud
  resourceVersion: "769"
  uid: 451817ab-9544-4e97-b7ff-6eddbc5928b5
  generation: 1
  creationTimestamp: "2026-06-09T08:06:00Z"
  managedFields:
    - manager: kubectl
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {}
spec:
  group: flux.services.openmcp.cloud
status:
  acceptedNames:
    kind: Flux
`
	out, err := stripCRDMetadata([]byte(input))
	if err != nil {
		t.Fatalf("stripCRDMetadata() error = %v", err)
	}
	result := string(out)

	stripped := []string{"resourceVersion", "uid", "generation", "creationTimestamp", "managedFields", "annotations", "status"}
	for _, field := range stripped {
		if strings.Contains(result, field) {
			t.Errorf("expected %q to be stripped, but found it in output:\n%s", field, result)
		}
	}

	kept := []string{"apiVersion", "kind", "fluxes.flux.services.openmcp.cloud", "spec", "flux.services.openmcp.cloud"}
	for _, field := range kept {
		if !strings.Contains(result, field) {
			t.Errorf("expected %q to be preserved, but not found in output:\n%s", field, result)
		}
	}
}

func TestStripCRDMetadataInvalidYAML(t *testing.T) {
	_, err := stripCRDMetadata([]byte("{{invalid yaml{{"))
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestRewriteKubeconfigServer(t *testing.T) {
	tests := []struct {
		name      string
		kubeconfig string
		oldServer  string
		nodePort   string
		wantOld    string
		wantNew    string
	}{
		{
			name: "rewrites localhost:8443",
			kubeconfig: `
clusters:
- cluster:
    server: https://localhost:8443
  name: kcp
`,
			oldServer: "https://localhost:8443",
			nodePort:  "32443",
			wantOld:   "https://localhost:8443",
			wantNew:   "https://localhost:32443",
		},
		{
			name: "rewrites all occurrences",
			kubeconfig: `
clusters:
- cluster:
    server: https://localhost:8443
- cluster:
    server: https://localhost:8443
`,
			oldServer: "https://localhost:8443",
			nodePort:  "30001",
			wantOld:   "https://localhost:8443",
			wantNew:   "https://localhost:30001",
		},
		{
			name: "no match leaves content unchanged",
			kubeconfig: `
clusters:
- cluster:
    server: https://some-other-host:6443
`,
			oldServer: "https://localhost:8443",
			nodePort:  "32443",
			wantOld:   "",
			wantNew:   "https://some-other-host:6443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteKubeconfigServer(tt.kubeconfig, tt.oldServer, tt.nodePort)
			if tt.wantOld != "" && strings.Contains(got, tt.wantOld) {
				t.Errorf("old server address %q still present in output", tt.wantOld)
			}
			if !strings.Contains(got, tt.wantNew) {
				t.Errorf("expected %q in output, got:\n%s", tt.wantNew, got)
			}
		})
	}
}

func TestKCPServerFromKubeconfig(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		wantServer  string
		wantErrFrag string
	}{
		{
			name: "reads server from current context",
			kubeconfig: `
apiVersion: v1
kind: Config
current-context: kcp
contexts:
- context:
    cluster: kcp-cluster
    user: admin
  name: kcp
clusters:
- cluster:
    server: https://localhost:8443
  name: kcp-cluster
users:
- name: admin
`,
			wantServer: "https://localhost:8443",
		},
		{
			name: "strips path from server URL",
			kubeconfig: `
apiVersion: v1
kind: Config
current-context: kcp
contexts:
- context:
    cluster: kcp-cluster
    user: admin
  name: kcp
clusters:
- cluster:
    server: https://localhost:8443/clusters/root
  name: kcp-cluster
users:
- name: admin
`,
			wantServer: "https://localhost:8443",
		},
		{
			name: "no current-context returns error",
			kubeconfig: `
apiVersion: v1
kind: Config
contexts: []
clusters: []
users: []
`,
			wantErrFrag: "no current-context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "kubeconfig-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(f.Name()) }()
			if _, err := f.WriteString(tt.kubeconfig); err != nil {
				t.Fatal(err)
			}
			_ = f.Close()

			got, err := kcpServerFromKubeconfig(f.Name())
			if tt.wantErrFrag != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrFrag) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErrFrag, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantServer {
				t.Errorf("got %q, want %q", got, tt.wantServer)
			}
		})
	}
}
