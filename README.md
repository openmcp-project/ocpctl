# oink 🐷 - cute openMCP environments in kind

oink is a tool for running local [openMCP](https://github.com/openmcp-project) environments in [kind](https://kind.sigs.k8s.io/) clusters. oink may be used for local development or CI.

## Run

```
$ go run cmd/oink/main.go

[Run] Create kind cluster oink.dev.platform
[Run] Export kubeconfig for kind cluster oink.dev.platform
[Run] Create system namespace
[Run] Create ServiceAccount for openmcp-operator
[Run] Create ClusterRoleBinding for openmcp-operator
[Run] Create ConfigMap for openmcp-operator
[Run] Create Deployment for openmcp-operator
```
