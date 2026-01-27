# GPU Operator Kyma Module - Quick Start

This guide helps you get the GPU Operator module running in your Kyma cluster quickly.

## TL;DR

```bash
# 1. Set up GPU worker pool in SAP BTP cockpit
#    - Add worker pool named "gpu"
#    - Use g6.xlarge or similar GPU nodes
#    - Set min=0, max=2 for auto-scaling

# 2. Install the module CRDs
make install

# 3. Deploy the controller
make deploy IMG=your-registry/gpu-operator:latest

# 4. Create a GpuOperator resource
kubectl apply -f config/samples/operator_v1alpha1_gpuoperator.yaml

# 5. Verify installation
kubectl get gpuoperator -A
kubectl get pods -n gpu-operator

# 6. Test GPU access
kubectl apply -f - <<EOF
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

kubectl wait --for=jsonpath='{.status.phase}'=Succeeded pod/gpu-test --timeout=300s
kubectl logs gpu-test
```

## What Gets Installed

The GPU Operator module automatically:

1. Creates the `gpu-operator` namespace
2. Installs NVIDIA GPU Operator components:
   - Driver DaemonSet (version 570 for Garden Linux)
   - Device Plugin DaemonSet
   - DCGM Exporter for monitoring
   - GPU Feature Discovery
3. Configures resources to advertise `nvidia.com/gpu`
4. Sets up proper RBAC and service accounts

## Next Steps

- **Monitor GPU nodes**: `kubectl get nodes -l workload=gpu`
- **Check autoscaler**: `kubectl get configmap -n kube-system cluster-autoscaler-status -o yaml`
- **Deploy AI workloads**: Use `nvidia.com/gpu: 1` in resource limits
- **Try the demo**: Deploy Stable Diffusion XL with Fooocus (see README)

## Using in Another Module

If you're building an LLM or AI module that needs GPU support:

1. Add this module as a prerequisite in your module config
2. In your workload specs, request GPU resources:
   ```yaml
   resources:
     limits:
       nvidia.com/gpu: 1  # Request 1 GPU
   ```
3. The GPU Operator module handles all the infrastructure setup

## Common Issues

**Pods stuck in Pending**: GPU nodes are being provisioned, wait ~5 minutes

**No GPU nodes**: Check SAP BTP cockpit worker pool configuration

**Driver installation fails**: Ensure driver version 570 is set (default)

For detailed troubleshooting, see the main [README.md](README.md#troubleshooting).

## Development Mode

Build and test locally:

```bash
# Run controller locally (requires cluster access)
make install
make run

# In another terminal, create a GpuOperator CR
kubectl apply -f config/samples/operator_v1alpha1_gpuoperator.yaml
```

## Building the Module Bundle

```bash
# Generate manifests
make build-manifests

# Create module with modulectl
modulectl create --insecure \
  --registry your-registry.io/unsigned \
  --module-config-file module-config.yaml

# This generates template.yaml with the ModuleTemplate CR
```

## Resources

- [Full Documentation](README.md)
- [SAP Kyma GPU Sample](https://github.com/SAP-samples/kyma-runtime-samples/blob/main/gpu/README.md)
- [NVIDIA GPU Operator Docs](https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/)
- [Kyma Modules Guide](https://github.com/kyma-project/template-operator)
