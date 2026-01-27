# GPU Operator Kyma Module

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/gpu-operator)](https://api.reuse.software/info/github.com/kyma-project/gpu-operator)

## Overview

The GPU Operator Kyma module enables GPU workloads in Kyma clusters by automating the installation and management of NVIDIA GPU Operator. This module is specifically designed to work with Garden Linux-based Kyma clusters and provides a declarative way to enable GPU support.

This module wraps the NVIDIA GPU Operator installation as described in the [SAP Kyma Runtime GPU Sample](https://github.com/SAP-samples/kyma-runtime-samples/blob/main/gpu/README.md), making it easy to use GPU capabilities in other modules (such as LLM deployment modules).

## Features

- **Declarative GPU Operator Management**: Install and configure NVIDIA GPU Operator using Kubernetes Custom Resources
- **Garden Linux Optimized**: Pre-configured values for Garden Linux kernel compatibility (driver version 570)
- **Kyma Integration**: Follows Kyma module conventions with proper state management and conditions
- **Automatic Resource Management**: Handles namespace creation, RBAC, and cleanup
- **Customizable Configuration**: Support for custom Helm values via ConfigMaps
- **Status Reporting**: Clear status conditions indicating the health of GPU operator installation

## Prerequisites

### Cluster Requirements

- SAP BTP Kyma runtime instance
- GPU worker pool configured with GPU-enabled nodes (e.g., `g6.xlarge`)
- Kubernetes 1.27+ (compatible with Kyma runtime)
- kubectl configured to access your cluster

### GPU Worker Pool Setup

Follow these steps to set up a GPU worker pool in your Kyma cluster:

1. Go to SAP BTP cockpit and update your Kyma instance
2. Add a new worker pool named `gpu`
3. Add nodes with GPU support (e.g., `g6.xlarge`)
4. Set auto-scaling:
   - Min nodes: `0` (to save costs when no GPU workloads are running)
   - Max nodes: `2` (or desired number)

For more information, see [Additional Worker Node Pools](https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?version=Cloud#additional-worker-node-pools).

### Tools

- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [kubebuilder](https://book.kubebuilder.io/) (for development)
- [modulectl](https://github.com/kyma-project/modulectl/releases) (for module packaging)
- [Helm 3.x](https://helm.sh/docs/intro/install/) (installed on cluster or locally)

## Installation

### Option 1: Using Kyma Lifecycle Manager (Recommended)

1. Apply the ModuleTemplate to your Kyma Control Plane (generated via `modulectl`)

2. Enable the module in your Kyma CR:

```yaml
apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: my-kyma
  namespace: kcp-system
spec:
  modules:
    - name: gpu-operator
      channel: regular
```

3. Create a GpuOperator CR in your cluster:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: GpuOperator
metadata:
  name: gpu-operator
  namespace: default
spec:
  driverVersion: "570"
  namespace: gpu-operator
```

### Option 2: Direct Installation (Development)

1. Install the CRD:

```bash
make install
```

2. Deploy the controller:

```bash
make deploy IMG=<your-registry>/gpu-operator:latest
```

3. Create a GpuOperator CR:

```bash
kubectl apply -f config/samples/operator_v1alpha1_gpuoperator.yaml
```

## Usage

### Basic Configuration

Create a GpuOperator custom resource with default settings:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: GpuOperator
metadata:
  name: my-gpu-operator
  namespace: default
spec:
  driverVersion: "570"  # Compatible with Garden Linux
  namespace: gpu-operator
```

### Custom Helm Values

To use custom NVIDIA GPU Operator Helm values:

1. Create a ConfigMap with your custom values:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-gpu-values
  namespace: default
data:
  values.yaml: |
    driver:
      version: "570"
      usePrecompiled: true
    devicePlugin:
      enabled: true
    # ... other custom values
```

2. Reference the ConfigMap in your GpuOperator CR:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: GpuOperator
metadata:
  name: my-gpu-operator
  namespace: default
spec:
  driverVersion: "570"
  namespace: gpu-operator
  valuesConfigMapName: custom-gpu-values
```

### Resource Requirements

Specify resource limits for GPU operator components:

```yaml
apiVersion: operator.kyma-project.io/v1alpha1
kind: GpuOperator
metadata:
  name: my-gpu-operator
  namespace: default
spec:
  driverVersion: "570"
  namespace: gpu-operator
  resources:
    limits:
      cpu: "1"
      memory: 1Gi
    requests:
      cpu: 100m
      memory: 256Mi
```

## Verification

### Check Module Status

```bash
kubectl get gpuoperator -A
```

Expected output:
```
NAME              STATE   DRIVER VERSION   AGE
my-gpu-operator   Ready   570              5m
```

### Check GPU Operator Pods

```bash
kubectl get pods -n gpu-operator
```

You should see pods for:
- GPU operator controller
- NVIDIA driver daemonset
- Device plugin daemonset
- DCGM exporter
- GPU feature discovery

### Test GPU Access

Deploy a test workload:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: gpu-test
spec:
  containers:
  - name: gpu-test
    image: nvcr.io/nvidia/cuda:13.0.1-runtime-ubuntu24.04
    command: ["nvidia-smi"]
    resources:
      limits:
        nvidia.com/gpu: 1
  restartPolicy: Never
EOF
```

Wait for the pod to complete and check the output:

```bash
kubectl wait --for=jsonpath='{.status.phase}'=Succeeded pod/gpu-test --timeout=300s
kubectl logs gpu-test
```

You should see NVIDIA GPU information from `nvidia-smi`.

### Check Cluster Autoscaler

Monitor GPU node provisioning:

```bash
kubectl get configmap -n kube-system cluster-autoscaler-status -o yaml
```

Look for your GPU worker pool in the output.

## Advanced Usage

### Using with LLM Deployment Modules

This module is designed to be a dependency for LLM deployment modules. Once installed, other modules can schedule GPU workloads by requesting `nvidia.com/gpu` resources:

```yaml
resources:
  limits:
    nvidia.com/gpu: 1
```

### Demo: AI Image Generation

For a more impressive demonstration using Stable Diffusion XL:

```bash
kubectl apply -f https://raw.githubusercontent.com/SAP-samples/kyma-runtime-samples/main/gpu/fooocus.yaml
```

Access the web UI via your Kyma domain: `https://fooocus.<your-cluster-domain>.kyma.ondemand.com/`

Clean up:
```bash
kubectl delete -f https://raw.githubusercontent.com/SAP-samples/kyma-runtime-samples/main/gpu/fooocus.yaml
```

## Monitoring

The module exposes Prometheus metrics for monitoring GPU operator health:

- Pod status and readiness
- Resource utilization
- Driver installation status

Access metrics via the controller's metrics endpoint on port 8443.

## Troubleshooting

### GPU Operator Not Ready

Check the GpuOperator status conditions:

```bash
kubectl get gpuoperator my-gpu-operator -o yaml
```

Look at the `status.conditions` section for detailed error messages.

### No GPU Nodes Available

Verify that GPU nodes are being provisioned:

```bash
kubectl get nodes -l workload=gpu
```

If no nodes appear, check:
1. GPU worker pool configuration in SAP BTP cockpit
2. Cluster autoscaler status
3. Node taints and labels

### Driver Installation Failed

Check driver daemonset logs:

```bash
kubectl logs -n gpu-operator -l app=nvidia-driver-daemonset
```

Common issues:
- Kernel version incompatibility (use driver version 570 for Garden Linux)
- Missing kernel headers
- Insufficient node resources

### Pods Stuck in Pending

If GPU pods remain in `Pending` state:

1. Check if `nvidia.com/gpu` resources are advertised:
```bash
kubectl describe node <gpu-node-name>
```

2. Verify device plugin is running:
```bash
kubectl get pods -n gpu-operator -l app=nvidia-device-plugin-daemonset
```

3. Check for node taints that might block scheduling

## Development

### Prerequisites

- Go 1.23+
- kubebuilder 3.x
- Docker or Podman
- Access to a Kubernetes cluster

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Build and push Docker image
make docker-build docker-push IMG=<registry>/gpu-operator:<tag>

# Generate manifests
make manifests

# Generate code
make generate
```

### Local Development

Run the controller locally:

```bash
# Install CRDs
make install

# Run controller locally
make run
```

### Creating a Module

Build and push the module to your registry:

```bash
# Build manifests
make build-manifests

# Create module using modulectl
modulectl create --insecure \
  --registry <your-registry>/unsigned \
  --module-config-file module-config.yaml
```

This generates a `template.yaml` file containing the ModuleTemplate CR.

## Architecture

### Components

- **GpuOperator Controller**: Reconciles GpuOperator CRs and manages NVIDIA GPU Operator installation
- **Module Data**: Contains pre-configured YAML manifests and Helm values
- **CRD**: Defines the GpuOperator custom resource schema

### State Management

The module follows Kyma state management conventions:

- `Processing`: Installation or update in progress
- `Ready`: GPU Operator successfully installed and running
- `Error`: Installation or reconciliation failed
- `Deleting`: Cleanup in progress

### Conditions

The module reports these conditions:

- `Ready`: Overall readiness of GPU operator
- `Installed`: Whether GPU operator resources are installed

## Configuration Reference

### GpuOperatorSpec

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `driverVersion` | string | NVIDIA driver version | `"570"` |
| `namespace` | string | Installation namespace | `"gpu-operator"` |
| `valuesConfigMapName` | string | ConfigMap with custom Helm values | - |
| `resources` | object | Resource requirements | - |

### GpuOperatorStatus

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | Current state (Ready, Processing, Error, Deleting) |
| `conditions` | array | Detailed status conditions |
| `installedVersion` | string | Installed driver version |
| `observedGeneration` | int64 | Last processed generation |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to this module.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## References

- [NVIDIA GPU Operator](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/overview.html)
- [Kyma GPU Runtime Sample](https://github.com/SAP-samples/kyma-runtime-samples/blob/main/gpu/README.md)
- [Kyma Modularization](https://github.com/kyma-project/community/tree/main/concepts/modularization)
- [Gardener AI Conformance](https://github.com/gardener/gardener-ai-conformance)
- [Garden Linux GPU Operator Guide](https://github.com/gardener/gardener-ai-conformance/blob/main/v1.33/NVIDIA-GPU-Operator.md)
