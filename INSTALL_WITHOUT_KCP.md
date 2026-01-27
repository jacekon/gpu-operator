# Installing GPU Operator Module Without KCP Access

When you don't have access to the Kyma Control Plane (KCP), you can't modify the ModuleTemplate catalog. However, you can still install the GPU Operator module directly on your cluster using one of these approaches:

## Approach 1: Direct Installation (Fastest - No Docker Build Required)

This approach installs just the CRD and runs the controller locally, which is perfect for development and testing.

### Steps:

1. **Install the CRD only:**
```bash
export KUBECONFIG=~/kyma-modules/llm-manager/kubeconfig_gpus.yaml
kubectl apply -f config/crd/bases/operator.kyma-project.io_gpuoperators.yaml
```

2. **Run the controller locally** (in a separate terminal):
```bash
export KUBECONFIG=~/kyma-modules/llm-manager/kubeconfig_gpus.yaml
cd /Users/D061255/kyma-modules/gpu-operator
make run
```

3. **Create a GpuOperator CR:**
```bash
kubectl apply -f config/samples/operator_v1alpha1_gpuoperator.yaml
```

## Approach 2: Full Cluster Installation (Production-like)

This approach deploys the controller as a Deployment in the cluster.

### Prerequisites:
- Docker or Podman configured and running
- Access to push images to a container registry (Docker Hub, GCR, etc.)

### Steps:

1. **Build and push the controller image:**
```bash
# Login to your container registry
docker login

# Build and push
cd /Users/D061255/kyma-modules/gpu-operator
make docker-build docker-push IMG=<your-registry>/gpu-operator:latest
```

2. **Update the manifest to use your image:**
```bash
cd config/manager
kustomize edit set image controller=<your-registry>/gpu-operator:latest
cd ../..
make build-manifests
```

3. **Install on cluster:**
```bash
export KUBECONFIG=~/kyma-modules/llm-manager/kubeconfig_gpus.yaml
kubectl apply -f gpu-operator.yaml
```

4. **Create a GpuOperator CR:**
```bash
kubectl apply -f config/samples/operator_v1alpha1_gpuoperator.yaml
```

## Approach 3: Using pre-built manifests (Current Situation)

If you've already applied `gpu-operator.yaml` but the image pull is failing:

1. **Check the current deployment:**
```bash
export KUBECONFIG=~/kyma-modules/llm-manager/kubeconfig_gpus.yaml
kubectl get deploy -n gpu-operator-system
kubectl describe pod -n gpu-operator-system
```

2. **Option A: Run controller locally instead**
   - Delete the deployment: `kubectl delete deploy gpu-operator-controller-manager -n gpu-operator-system`
   - Run locally: `make run KUBECONFIG=~/kyma-modules/llm-manager/kubeconfig_gpus.yaml`

3. **Option B: Build and update the image**
   - Fix Docker credentials issue
   - Build and push image
   - Update deployment: `kubectl set image deployment/gpu-operator-controller-manager -n gpu-operator-system manager=<your-image>`

## Verifying Installation

```bash
# Check if CRD is installed
kubectl get crd gpuoperators.operator.kyma-project.io

# Check controller (if running in cluster)
kubectl get pods -n gpu-operator-system

# Create a test GpuOperator
cat <<EOF | kubectl apply -f -
apiVersion: operator.kyma-project.io/v1alpha1
kind: GpuOperator
metadata:
  name: test-gpu-operator
  namespace: default
spec:
  driverVersion: "570"
  namespace: gpu-operator
EOF

# Watch the installation
kubectl get gpuoperator -A -w
```

## Current Status

Based on your current situation:
- ✅ CRD is installed
- ✅ RBAC is configured  
- ❌ Controller pod is failing with `ErrImagePull` because the image `controller:latest` doesn't exist

**Recommended Next Step:** Use Approach 1 (run controller locally) to test immediately without dealing with Docker/registry issues.

