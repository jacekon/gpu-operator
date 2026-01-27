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

// State is a string type that represents the state of the module.
type State string

const (
	// StateReady signifies that the module is installed and ready.
	StateReady State = "Ready"

	// StateProcessing signifies that the module is being processed.
	StateProcessing State = "Processing"

	// StateError signifies that the module is in an error state.
	StateError State = "Error"

	// StateDeleting signifies that the module is being deleted.
	StateDeleting State = "Deleting"
)

// Status defines the observed state of Module CR.
type Status struct {
	// State signifies current state of Module CR.
	// Value can be one of ("Ready", "Processing", "Error", "Deleting").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State State `json:"state"`
}
