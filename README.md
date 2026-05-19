# ocpctl

ocpctl is a CLI tool for [OpenControlPlane](https://openmcp-project.github.io/docs/).

> **Note:** ocpctl currently supports managing local environments running in [kind](https://kind.sigs.k8s.io/) clusters, for development, testing, and CI use cases. Additional features are planned for future releases.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)

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

### List environments

Lists all ocpctl-managed local environments:

```bash
ocpctl env list
```

Only environments whose platform cluster has the Cluster CRD installed (i.e. were created by ocpctl) are shown.

### Delete an environment

Deletes all kind clusters belonging to an environment:

```bash
ocpctl env delete my-env
```

This connects to the platform cluster, collects every kind cluster recorded in the `Cluster` resource provider statuses, and deletes them all.

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
      image: ghcr.io/openmcp-project/images/cluster-provider-kind:v0.3.1
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

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/ocpctl/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](https://github.com/openmcp-project/.github/blob/main/CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/openmcp-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing
Copyright © Linux Foundation Europe. OpenControlPlane is a project of NeoNephos Foundation. For applicable policies including privacy policy, terms of use and trademark usage guidelines, please see https://linuxfoundation.eu. Linux is a registered trademark of Linus Torvalds.
Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/ocpctl).

<p align="center"><img alt="NeoNephos foundation logo" src="https://raw.githubusercontent.com/neonephos/.github/refs/heads/main/assets/logo.svg" width="400"/></p>
