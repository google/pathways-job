# pathways-job

![Version: 0.1.2](https://img.shields.io/badge/Version-0.1.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

A Helm chart for deploying Pathways controller and webhook on Kubernetes.

**Homepage:** <https://github.com/google/pathways-job>

## Introduction

This Helm chart installs the Pathways-Job controller and webhook to your Kubernetes cluster. Pathways is a system designed to enable the creation of large-scale, multi-task, and sparsely activated machine learning systems.

## Prerequisites

- Helm >= 3
- Kubernetes >= 1.11.3
- JobSet >= 0.8.0

## Usage

### Install the chart

```shell
helm install [RELEASE_NAME] charts/pathways-job \
    --namespace pathways-job-system \
    --create-namespace
```

For example, if you want to create a release with name `pathways-job` in the `pathways-job-system` namespace:

```shell
helm install pathways-job charts/pathways-job \
    --namespace pathways-job-system \
    --create-namespace
```

Note that by passing the `--create-namespace` flag to the `helm install` command, `helm` will create the release namespace if it does not exist.

See [helm install](https://helm.sh/docs/helm/helm_install) for command documentation.

### Upgrade the chart

```shell
helm upgrade [RELEASE_NAME] charts/pathways-job [flags]
```

See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade) for command documentation.

### Uninstall the chart

```shell
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes resources associated with the chart and deletes the release.

See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall) for command documentation.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| nameOverride | string | `""` | String to partially override release name. |
| fullnameOverride | string | `""` | String to fully override release name. |
| image.repository | string | `"us-docker.pkg.dev/cloud-tpu-v2-images/pathways-job/pathwaysjob-controller"` | Image repository. |
| manager.replicas| int | `1` | Replicas for the controller manager. |
| manager.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":"ALL"}}` | Security context of all pathways-job controller containers. |
| manager.podSecurityContext | object | `{"runAsNonRoot":true,"seccompProfile":{"type":"RuntimeDefault"}}` | Security context of all pathways-job controller containers. |
| metrics.enable | bool| `false` | Whether to enable metrics exporting. |

## Maintainers

| Name | Url |
| ---- | --- |
| asall | <https://github.com/asall> |
