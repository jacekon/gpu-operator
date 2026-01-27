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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GpuOperatorSpec defines the desired state of GpuOperator
type GpuOperatorSpec struct {
	// DriverVersion specifies the NVIDIA driver version to install
	// Compatible with Garden Linux kernel versions in Kyma clusters
	// +optional
	// +kubebuilder:default="570"
	DriverVersion string `json:"driverVersion,omitempty"`

	// Namespace where the GPU operator will be installed
	// +optional
	// +kubebuilder:default="gpu-operator"
	Namespace string `json:"namespace,omitempty"`

	// ValuesConfigMapName is the name of the ConfigMap containing custom Helm values
	// If specified, these values will be used instead of the default values
	// +optional
	ValuesConfigMapName string `json:"valuesConfigMapName,omitempty"`

	// Resources defines resource limits for GPU operator components
	// +optional
	Resources *ResourceRequirements `json:"resources,omitempty"`
}

// ResourceRequirements defines CPU and memory requirements
type ResourceRequirements struct {
	// Limits defines the maximum resources for the operator
	// +optional
	Limits *Resources `json:"limits,omitempty"`

	// Requests defines the minimum resources for the operator
	// +optional
	Requests *Resources `json:"requests,omitempty"`
}

// Resources defines CPU and memory
type Resources struct {
	// CPU resource requirement
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource requirement
	// +optional
	Memory string `json:"memory,omitempty"`
}

// GpuOperatorStatus defines the observed state of GpuOperator
type GpuOperatorStatus struct {
	Status `json:",inline"`

	// Conditions contain a set of conditionals to determine the State of Status.
	// If all Conditions are met, State is expected to be in StateReady.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// InstalledVersion is the version of the GPU operator currently installed
	// +optional
	InstalledVersion string `json:"installedVersion,omitempty"`

	// ObservedGeneration is the generation of the GpuOperator CR that was last processed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Driver Version",type=string,JSONPath=`.spec.driverVersion`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// GpuOperator is the Schema for the gpuoperators API
type GpuOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GpuOperatorSpec   `json:"spec,omitempty"`
	Status GpuOperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GpuOperatorList contains a list of GpuOperator
type GpuOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GpuOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GpuOperator{}, &GpuOperatorList{})
}
