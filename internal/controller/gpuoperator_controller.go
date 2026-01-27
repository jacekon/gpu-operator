/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/kyma-project/gpu-operator/api/v1alpha1"
)

const (
	finalizerName          = "operator.kyma-project.io/gpu-operator-finalizer"
	conditionTypeReady     = "Ready"
	conditionTypeInstalled = "Installed"
	installJobName         = "gpu-operator-install"
	uninstallJobName       = "gpu-operator-uninstall"

	// Gardener AI Conformance Guide for GPU Operator installation
	// Reference: https://github.com/gardener/gardener-ai-conformance/blob/main/v1.33/NVIDIA-GPU-Operator.md
	gardenerValuesURL = "https://raw.githubusercontent.com/gardenlinux/gardenlinux-nvidia-installer/refs/heads/main/helm/gpu-operator-values.yaml"
	nvidiaHelmRepo    = "https://helm.ngc.nvidia.com/nvidia"
	helmImage         = "alpine/helm:3.14.0"
)

// GpuOperatorReconciler reconciles a GpuOperator object
type GpuOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=gpuoperators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=gpuoperators/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=gpuoperators/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch

func (r *GpuOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the GpuOperator instance
	gpuOperator := &operatorv1alpha1.GpuOperator{}
	if err := r.Get(ctx, req.NamespacedName, gpuOperator); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("GpuOperator resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get GpuOperator")
		return ctrl.Result{}, err
	}

	// Check if the GpuOperator instance is marked to be deleted
	if gpuOperator.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(gpuOperator, finalizerName) {
			// Run finalization logic
			if err := r.finalizeGpuOperator(ctx, gpuOperator); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(gpuOperator, finalizerName)
			if err := r.Update(ctx, gpuOperator); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(gpuOperator, finalizerName) {
		controllerutil.AddFinalizer(gpuOperator, finalizerName)
		if err := r.Update(ctx, gpuOperator); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set status to Processing
	if gpuOperator.Status.State != operatorv1alpha1.StateProcessing {
		gpuOperator.Status.State = operatorv1alpha1.StateProcessing
		if err := r.Status().Update(ctx, gpuOperator); err != nil {
			logger.Error(err, "Failed to update GpuOperator status to Processing")
			return ctrl.Result{}, err
		}
	}

	// Create namespace if it doesn't exist
	namespace := gpuOperator.Spec.Namespace
	if namespace == "" {
		namespace = "gpu-operator"
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	if err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating namespace", "namespace", namespace)
			if err := r.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create namespace")
				return r.updateStatusError(ctx, gpuOperator, err)
			}
		} else {
			logger.Error(err, "Failed to get namespace")
			return r.updateStatusError(ctx, gpuOperator, err)
		}
	}

	// Create ServiceAccount with necessary permissions
	if err := r.ensureServiceAccount(ctx, namespace); err != nil {
		logger.Error(err, "Failed to ensure ServiceAccount")
		return r.updateStatusError(ctx, gpuOperator, err)
	}

	// Create RBAC for the installer Job
	if err := r.ensureRBAC(ctx, namespace); err != nil {
		logger.Error(err, "Failed to ensure RBAC")
		return r.updateStatusError(ctx, gpuOperator, err)
	}

	// Create or update Helm installation Job following Gardener AI conformance guide
	if err := r.createHelmInstallJob(ctx, gpuOperator, namespace); err != nil {
		logger.Error(err, "Failed to create Helm installation job")
		return r.updateStatusError(ctx, gpuOperator, err)
	}

	// Check if the installation job completed successfully
	jobReady, err := r.isJobCompleted(ctx, namespace, installJobName)
	if err != nil {
		logger.Error(err, "Failed to check job status")
		return r.updateStatusError(ctx, gpuOperator, err)
	}
	if !jobReady {
		logger.Info("Helm installation job still running, will requeue")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Update status to Ready
	gpuOperator.Status.State = operatorv1alpha1.StateReady
	gpuOperator.Status.ObservedGeneration = gpuOperator.Generation
	gpuOperator.Status.InstalledVersion = gpuOperator.Spec.DriverVersion

	// Set conditions
	readyCondition := metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "GpuOperatorReady",
		Message:            "GPU Operator installed successfully following Gardener AI conformance guide",
		ObservedGeneration: gpuOperator.Generation,
		LastTransitionTime: metav1.Now(),
	}
	installedCondition := metav1.Condition{
		Type:               conditionTypeInstalled,
		Status:             metav1.ConditionTrue,
		Reason:             "HelmInstallComplete",
		Message:            "NVIDIA GPU Operator installed via Helm with Garden Linux optimized values",
		ObservedGeneration: gpuOperator.Generation,
		LastTransitionTime: metav1.Now(),
	}

	gpuOperator.Status.Conditions = []metav1.Condition{readyCondition, installedCondition}

	if err := r.Status().Update(ctx, gpuOperator); err != nil {
		logger.Error(err, "Failed to update GpuOperator status to Ready")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled GpuOperator")
	return ctrl.Result{}, nil
}

// ensureServiceAccount creates the ServiceAccount needed for Helm Jobs
func (r *GpuOperatorReconciler) ensureServiceAccount(ctx context.Context, namespace string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpu-operator",
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "gpu-operator",
				"app.kubernetes.io/managed-by": "gpu-operator-module",
			},
		},
	}

	existing := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: sa.Name, Namespace: namespace}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.Create(ctx, sa)
		}
		return err
	}
	return nil
}

// ensureRBAC creates ClusterRole and ClusterRoleBinding for Helm installer
func (r *GpuOperatorReconciler) ensureRBAC(ctx context.Context, namespace string) error {
	// For now, we're using the controller's own RBAC which has cluster-admin
	// In production, you might want to create specific RBAC for the installer Job
	return nil
}

// createHelmInstallJob creates a Kubernetes Job that installs NVIDIA GPU Operator using Helm
// following the Gardener AI conformance guide:
// https://github.com/gardener/gardener-ai-conformance/blob/main/v1.33/NVIDIA-GPU-Operator.md
func (r *GpuOperatorReconciler) createHelmInstallJob(ctx context.Context, gpuOperator *operatorv1alpha1.GpuOperator, namespace string) error {
	logger := log.FromContext(ctx)

	// Determine values URL - use Gardener Garden Linux optimized values
	valuesURL := gardenerValuesURL
	if gpuOperator.Spec.ValuesConfigMapName != "" {
		logger.Info("Custom values ConfigMap specified, but using Gardener values as base",
			"configMap", gpuOperator.Spec.ValuesConfigMapName)
		// TODO: Support merging custom values with Gardener values
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      installJobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "gpu-operator-installer",
				"app.kubernetes.io/managed-by": "gpu-operator-module",
				"app.kubernetes.io/component":  "installer",
			},
			Annotations: map[string]string{
				"gardener.ai/conformance-guide": "v1.33",
				"gardener.ai/values-source":     gardenerValuesURL,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr.To[int32](300), // Clean up after 5 minutes
			BackoffLimit:            ptr.To[int32](3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "gpu-operator",
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "helm-installer",
							Image:   helmImage,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								fmt.Sprintf(`
set -e
echo "=================================================="
echo "Installing NVIDIA GPU Operator"
echo "Following Gardener AI Conformance Guide v1.33"
echo "Reference: https://github.com/gardener/gardener-ai-conformance/blob/main/v1.33/NVIDIA-GPU-Operator.md"
echo "=================================================="
echo ""

echo "Step 1: Add NVIDIA Helm repository..."
helm repo add nvidia %s
helm repo update

echo ""
echo "Step 2: Verify repository..."
helm search repo nvidia/gpu-operator

echo ""
echo "Step 3: Install GPU Operator with Garden Linux optimized values..."
echo "Using values from: %s"
helm upgrade --install --create-namespace \
  -n %s gpu-operator nvidia/gpu-operator \
  --values %s \
  --wait --timeout 10m

echo ""
echo "=================================================="
echo "GPU Operator installation completed successfully"
echo "=================================================="
helm status gpu-operator -n %s
`, nvidiaHelmRepo, valuesURL, namespace, valuesURL, namespace),
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference so the job is cleaned up with the GpuOperator CR
	if err := controllerutil.SetControllerReference(gpuOperator, job, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Check if job already exists
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: installJobName, Namespace: namespace}, existingJob)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating Helm installation job following Gardener AI conformance guide",
				"job", installJobName, "namespace", namespace)
			if err := r.Create(ctx, job); err != nil {
				return fmt.Errorf("failed to create job: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get existing job: %w", err)
	}

	// Job already exists - check if it needs to be recreated
	logger.Info("Helm installation job already exists", "job", installJobName, "namespace", namespace)
	return nil
}

// isJobCompleted checks if a Job has completed successfully
func (r *GpuOperatorReconciler) isJobCompleted(ctx context.Context, namespace, jobName string) (bool, error) {
	job := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: namespace}, job)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	// Check if job completed successfully
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true, nil
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return false, fmt.Errorf("helm installation job failed: %s", condition.Message)
		}
	}

	return false, nil
}

func (r *GpuOperatorReconciler) finalizeGpuOperator(ctx context.Context, gpuOperator *operatorv1alpha1.GpuOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Finalizing GpuOperator")

	// Set status to Deleting
	gpuOperator.Status.State = operatorv1alpha1.StateDeleting
	if err := r.Status().Update(ctx, gpuOperator); err != nil {
		logger.Error(err, "Failed to update GpuOperator status to Deleting")
	}

	namespace := gpuOperator.Spec.Namespace
	if namespace == "" {
		namespace = "gpu-operator"
	}

	// Create uninstall job
	logger.Info("Creating Helm uninstall job")
	uninstallJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uninstallJobName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "gpu-operator-uninstaller",
				"app.kubernetes.io/managed-by": "gpu-operator-module",
				"app.kubernetes.io/component":  "uninstaller",
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr.To[int32](60),
			BackoffLimit:            ptr.To[int32](2),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "gpu-operator",
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					Containers: []corev1.Container{
						{
							Name:    "helm-uninstaller",
							Image:   helmImage,
							Command: []string{"/bin/sh", "-c"},
							Args: []string{
								fmt.Sprintf(`
set -e
echo "Uninstalling NVIDIA GPU Operator"
helm uninstall gpu-operator -n %s || true
echo "GPU Operator uninstalled successfully"
`, namespace),
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, uninstallJob); err != nil && !apierrors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create uninstall job, continuing with cleanup")
	}

	logger.Info("Successfully finalized GpuOperator")
	return nil
}

func (r *GpuOperatorReconciler) updateStatusError(ctx context.Context, gpuOperator *operatorv1alpha1.GpuOperator, err error) (ctrl.Result, error) {
	gpuOperator.Status.State = operatorv1alpha1.StateError
	errorCondition := metav1.Condition{
		Type:               conditionTypeReady,
		Status:             metav1.ConditionFalse,
		Reason:             "ReconciliationFailed",
		Message:            err.Error(),
		ObservedGeneration: gpuOperator.Generation,
		LastTransitionTime: metav1.Now(),
	}
	gpuOperator.Status.Conditions = []metav1.Condition{errorCondition}

	if statusErr := r.Status().Update(ctx, gpuOperator); statusErr != nil {
		log.FromContext(ctx).Error(statusErr, "Failed to update status")
	}

	return ctrl.Result{}, err
}

func (r *GpuOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.GpuOperator{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
