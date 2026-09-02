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

ocpctl is organized into command groups, each managing a different aspect of your OpenControlPlane setup. The sections below describe the available commands and how to use them.

## Environments

```bash
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
  flux:
    namespace: "my-flux-namespace"
```

Only fields present in the config file override the defaults — omitted fields keep their default values.

#### Replacing provider lists

By default the `clusterProviders`, `serviceProviders`, and `platformServices`
lists are *merged* into the defaults by name, so an omitted or empty list keeps
all defaults. To install fewer components than the defaults, set the
corresponding `*Mode` field to `replace`; the config's list then wins wholesale,
including an empty list:

```yaml
spec:
  serviceProvidersMode: replace   # default: merge
  serviceProviders: []            # with mode=replace -> install none
```

The same applies to `clusterProvidersMode` and `platformServicesMode`. Validate
a config file with:

```bash
ocpctl env validate --config env.yaml
```

#### Flux Configuration

The following fields are provided to configure the deployed flux instance

| Field | Type | Default | Description |
|---|---|---|---|
| `version` | `string` | `latest` | Flux version to install |
| `namespace` | `string` | `flux-system` | Namespace into which Flux is installed |
| `registry` | `string` | `ghcr.io/fluxcd` | OCI registry from which Flux images are pulled |
| `registryCredential` | `string` | `""` | Credentials for the registry (`user:password`) |
| `imagePullSecret` | `string` | `""` | Name of an existing image pull secret in the Flux namespace |
| `baseURL` | `string` | `https://github.com/fluxcd/flux2/releases` | Base URL used to download Flux manifests |
| `logLevel` | `string` | `info` | Log level for all Flux controllers (`debug`, `info`, `error`) |
| `componentsExtra` | `[]string` | `[]` | Additional Flux controllers to install on top of the core set (default installs none) |
| `clusterDomain` | `string` | `cluster.local` | Kubernetes cluster domain used for internal DNS |
| `networkPolicy` | `bool` | `true` | Whether to install Flux's default NetworkPolicy resources |
| `tolerationKeys` | `[]string` | `[]` | Toleration keys added to all Flux controller deployments |

## Multiple environments

Each environment gets its own set of kind clusters prefixed with the environment name (e.g. `my-env-platform`), so multiple environments can coexist on the same machine.

## Clusters
```
ocpctl clusters list
ocpctl clusters kubeconfig get --environment <env> --name <clustername> [--internal]
ocpctl clusters kubeconfig export --environment <env> --name <clustername> [--kubeconfig <path-to-kubeconfig> --internal]
```

### List clusters

Lists all clusters across all managed environments:

```bash
ocpctl clusters list
```

### Get kubeconfig

Prints the kubeconfig for a specific cluster to stdout:

```bash
ocpctl clusters kubeconfig get --environment my-env --name my-cluster
```

Use `--internal` to get the kubeconfig with the address on the docker network.

### Export kubeconfig

Merges the kubeconfig for a cluster into a local kubeconfig file. The target file is resolved in order: `--kubeconfig` flag, then `$KUBECONFIG`, then `~/.kube/config`:

```bash
ocpctl clusters kubeconfig export --environment my-env --name my-cluster
```

Use `--kubeconfig` to target a different file and `--internal` to export the kubeconfig with the address on the docker network.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/ocpctl/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](https://github.com/openmcp-project/.github/blob/main/CONTRIBUTING.md).

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/openmcp-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright OpenControlPlane contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/ocpctl).

---

<p align="center">
  <a href="https://apeirora.eu/content/projects/">
    <img alt="BMWK-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="300"/>
  </a>
</p>

<p align="center">
  OpenControlPlane is part of <a href="https://apeirora.eu/content/projects/">ApeiroRA</a>, an EU Important Project of Common European Interest (IPCEI-CIS).
</p>

<p align="center">
  Copyright Linux Foundation Europe. For web site terms of use, trademark policy and other project policies please see <a href="https://linuxfoundation.eu/en/policies">https://linuxfoundation.eu/en/policies</a>.
</p>
