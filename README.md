# ocpctl

ocpctl is a CLI tool for [OpenControlPlane](https://openmcp-project.github.io/docs/).

> **Note:** ocpctl currently supports managing local environments running in [kind](https://kind.sigs.k8s.io/) clusters, for development, testing, and CI use cases. Additional features are planned for future releases.

## Prerequisites

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)

## Installation

```bash
go install github.com/openmcp-project/ocpctl@latest
```

## Usage

```
ocpctl env apply <name> [--config <file>]
ocpctl env delete <name>
ocpctl env list
```

### Apply an environment

Creates or updates a local OpenControlPlane environment:

```bash
ocpctl env apply my-env
```

This will:
1. Create a kind cluster named `my-env-platform` (if it doesn't exist)
2. Apply the openmcp-operator namespace, service account, RBAC, config, and deployment
3. Apply the ClusterProvider and PlatformCluster resources
4. Wait until all resources are ready

### Configuration

By default, ocpctl uses [built-in image versions](pkg/config/environment-defaults.yaml). To override them, provide a config file:

```bash
ocpctl env apply my-env --config env.yaml
```

Config files use the following format:

```yaml
apiVersion: ocpctl.open-control-plane.io/v1alpha1
kind: Environment
spec:
  namespace: openmcp-system
  operator:
    image: ghcr.io/openmcp-project/images/openmcp-operator:v0.18.1
  clusterProviders:
    - name: kind
      image: ghcr.io/openmcp-project/images/cluster-provider-kind:v0.2.0
  serviceProviders:
    - name: example
      image: ghcr.io/openmcp-project/images/service-provider-example:v0.4.1
  platformServices:
    - name: example
      image: ghcr.io/openmcp-project/images/platform-service-example:v0.0.10
```

Only fields present in the config file override the defaults — omitted fields keep their default values.

## Multiple environments

Each environment gets its own set of kind clusters prefixed with the environment name (e.g. `my-env-platform`), so multiple environments can coexist on the same machine.
