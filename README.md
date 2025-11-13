# openMCP in kind (omink)

omink is a tool for running local [openMCP](https://github.com/openmcp-project) environments in [kind](https://kind.sigs.k8s.io/) clusters. omink may be used for local development or CI.

## Run

```
$ go run cmd/omink/main.go

[Run] Create kind cluster omink.dev.platform
[Run] Export kubeconfig for kind cluster omink.dev.platform
[Run] Create system namespace
[Run] Create ServiceAccount for openmcp-operator
[Run] Create ClusterRoleBinding for openmcp-operator
[Run] Create ConfigMap for openmcp-operator
[Run] Create Deployment for openmcp-operator
```
