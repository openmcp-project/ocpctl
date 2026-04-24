# oink

oink is a CLI tool for running local [OpenControlPlane](https://openmcp-project.github.io/docs/) environments in [kind](https://kind.sigs.k8s.io/) clusters, for local development or CI.

## Prerequisites

- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)

## Installation

```bash
go install github.com/ValentinGerlach/oink@latest
```

## Usage

```
oink env create <name> [--operator-image <image>]
oink env update <name> [--operator-image <image>]
oink env delete <name>
oink env list
```

### Create an environment

```bash
oink env create my-env
```

This will:
1. Create a kind cluster named `my-env-platform`
2. Apply the openmcp-operator namespace, service account, RBAC, config, and deployment
3. Wait until all resources are ready

### Update an environment

Re-applies all platform resources against an existing cluster, useful for picking up image or config changes:

```bash
oink env update my-env --operator-image ghcr.io/openmcp-project/images/openmcp-operator:v0.19.0
```

## Multiple environments

Each environment gets its own set of kind clusters prefixed with the environment name (e.g. `my-env-platform`), so multiple environments can coexist on the same machine.
