# Jan Gateway Helm Chart

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v0.0.11](https://img.shields.io/badge/AppVersion-v0.0.11-informational?style=flat-square)

Helm chart for deploying the Jan Gateway with optional AI/ML inference support.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- Storage class with `ReadWriteMany` (required when inference is enabled)
- GPU nodes (optional, only required for GPU-backed inference)

## Quick Start

> **Default behavior:** `inference.enabled` is `true`. Installing the chart with default values will also install the GPU Operator, KubeRay Operator, Envoy Gateway, and Aibrix components, if they are already present in the cluster please skip each corresponding dependency by setting its toggle to `false`.

### Install without inference

```bash
helm install jan-server ./jan-server \
  --namespace jan-system \
  --create-namespace
```

### Install with inference enabled

```bash
helm install jan-server ./jan-server \
  --namespace jan-system \
  --create-namespace \
  --set inference.enabled=true \
  --set inference.dependencies.gpuOperator.enabled=true
```

## Values

| Key | Type | Default | Description |
| --- | ---- | ------- | ----------- |
| **Core** | | | |
| gateway.replicaCount | int | `1` | Number of application replicas |
| gateway.image.repository | string | `"janai/jan-server"` | Container image repository |
| gateway.image.tag | string | `""` | Image tag (defaults to chart appVersion) |
| **Dependencies** | | | |
| postgresql.enabled | bool | `true` | Deploy bundled PostgreSQL chart |
| valkey.enabled | bool | `true` | Deploy bundled Valkey (Redis) chart |
| **Inference** | | | |
| inference.enabled | bool | `false` | Toggle AI/ML inference components |
| inference.dependencies.gpuOperator.enabled | bool | `true` | Install NVIDIA GPU Operator when inference enabled |
| inference.dependencies.kuberayOperator.enabled | bool | `true` | Install KubeRay operator when inference enabled |
| inference.cleanup.autoCleanupDependencies | bool | `true` | Remove operator releases on uninstall |

> See `values.yaml` for the full list of configuration options.
>
> **Inference dependencies:** end users may install GPU Operator, KubeRay, Envoy Gateway, and Aibrix independently. When those operators are not yet present, turning the corresponding `inference.enabled` and dependency toggles to `true` triggers the chart hooks to install them automatically.

## Configuration Examples

### Minimal install

```yaml
inference:
  enabled: false

gateway:
  replicaCount: 1
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  env:
    - name: JAN_INFERENCE_MODEL_URL
      value: "https://your-external-inference-endpoint"
```

> When inference is disabled, update the `gateway.env` entry for `JAN_INFERENCE_MODEL_URL` to point at your own external model endpoint. The default value targets the in-cluster Envoy endpoint that is created when inference with Aibrix is enabled.

### Inference-enabled install

```yaml
inference:
  enabled: true
  storage:
    enabled: true
    storageClassName: "nfs-client"
    size: "200Gi"
  dependencies:
    gpuOperator:
      enabled: true
    kuberayOperator:
      enabled: true
  cleanup:
    autoCleanupDependencies: false
    cleanupGpuOperator: false
```

## Upgrade

```bash
helm upgrade jan-server ./jan-server \
  --namespace jan-system \
  --reuse-values
```

## Uninstall

```bash
helm uninstall jan-server --namespace jan-system

# enable cleanup if inference dependencies were installed
helm upgrade jan-server ./jan-server \
  --namespace jan-system \
  --set inference.cleanup.autoCleanupDependencies=true
helm uninstall jan-server --namespace jan-system
```

## Maintainers

| Name | Email | Url |
| --- | --- | --- |
| Jan.ai Team | <support@jan.ai> | <https://jan.ai> |

