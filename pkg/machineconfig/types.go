/*
Copyright 2024.

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

// Package machineconfig provides types for interacting with MachineConfig resources.
// These types are simplified versions to avoid importing the full machine-config-operator
// package which has compatibility issues on non-Linux systems.
package machineconfig

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the group version for MachineConfig resources.
var GroupVersion = schema.GroupVersion{Group: "machineconfiguration.openshift.io", Version: "v1"}

// MachineConfigPool is a simplified representation of the MachineConfigPool resource.
// +kubebuilder:object:root=true
type MachineConfigPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineConfigPoolSpec   `json:"spec,omitempty"`
	Status MachineConfigPoolStatus `json:"status,omitempty"`
}

// MachineConfigPoolSpec defines the spec of a MachineConfigPool.
type MachineConfigPoolSpec struct {
	// MachineConfigSelector selects which MachineConfigs to apply.
	MachineConfigSelector *metav1.LabelSelector `json:"machineConfigSelector,omitempty"`

	// NodeSelector selects which nodes belong to this pool.
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty"`

	// Paused specifies whether or not changes to this pool should be ignored.
	Paused bool `json:"paused,omitempty"`

	// MaxUnavailable specifies the maximum number of nodes that can be unavailable during update.
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`
}

// MachineConfigPoolStatus defines the status of a MachineConfigPool.
type MachineConfigPoolStatus struct {
	// ObservedGeneration is the generation observed by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Configuration contains the current rendered configuration.
	Configuration MachineConfigPoolStatusConfiguration `json:"configuration,omitempty"`

	// MachineCount is the total number of machines in the pool.
	MachineCount int32 `json:"machineCount,omitempty"`

	// UpdatedMachineCount is the number of machines that have been updated.
	UpdatedMachineCount int32 `json:"updatedMachineCount,omitempty"`

	// ReadyMachineCount is the number of ready machines.
	ReadyMachineCount int32 `json:"readyMachineCount,omitempty"`

	// UnavailableMachineCount is the number of unavailable machines.
	UnavailableMachineCount int32 `json:"unavailableMachineCount,omitempty"`

	// DegradedMachineCount is the number of degraded machines.
	DegradedMachineCount int32 `json:"degradedMachineCount,omitempty"`

	// Conditions represent the latest observations of the pool's state.
	Conditions []MachineConfigPoolCondition `json:"conditions,omitempty"`
}

// MachineConfigPoolStatusConfiguration contains the current rendered configuration.
type MachineConfigPoolStatusConfiguration struct {
	Name   string                          `json:"name,omitempty"`
	Source []MachineConfigPoolConfigSource `json:"source,omitempty"`
}

// MachineConfigPoolConfigSource identifies a MachineConfig.
type MachineConfigPoolConfigSource struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
}

// MachineConfigPoolCondition represents a condition of a MachineConfigPool.
type MachineConfigPoolCondition struct {
	Type               MachineConfigPoolConditionType `json:"type"`
	Status             string                         `json:"status"`
	LastTransitionTime metav1.Time                    `json:"lastTransitionTime,omitempty"`
	Reason             string                         `json:"reason,omitempty"`
	Message            string                         `json:"message,omitempty"`
}

// MachineConfigPoolConditionType are the types of conditions for MachineConfigPools.
type MachineConfigPoolConditionType string

const (
	// MachineConfigPoolUpdated indicates the pool has been updated.
	MachineConfigPoolUpdated MachineConfigPoolConditionType = "Updated"
	// MachineConfigPoolUpdating indicates the pool is updating.
	MachineConfigPoolUpdating MachineConfigPoolConditionType = "Updating"
	// MachineConfigPoolDegraded indicates the pool is degraded.
	MachineConfigPoolDegraded MachineConfigPoolConditionType = "Degraded"
	// MachineConfigPoolNodeDegraded indicates a node in the pool is degraded.
	MachineConfigPoolNodeDegraded MachineConfigPoolConditionType = "NodeDegraded"
	// MachineConfigPoolRenderDegraded indicates rendering is degraded.
	MachineConfigPoolRenderDegraded MachineConfigPoolConditionType = "RenderDegraded"
)

// MachineConfigPoolList contains a list of MachineConfigPools.
// +kubebuilder:object:root=true
type MachineConfigPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineConfigPool `json:"items"`
}

// MachineConfig is a simplified representation of the MachineConfig resource.
// +kubebuilder:object:root=true
type MachineConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MachineConfigSpec `json:"spec,omitempty"`
}

// MachineConfigSpec defines the spec of a MachineConfig.
type MachineConfigSpec struct {
	// OSImageURL is the URL of the OS image.
	OSImageURL string `json:"osImageURL,omitempty"`

	// Config is the Ignition config object.
	Config interface{} `json:"config,omitempty"`

	// KernelArguments is a list of kernel arguments.
	KernelArguments []string `json:"kernelArguments,omitempty"`

	// Extensions is a list of extensions.
	Extensions []string `json:"extensions,omitempty"`

	// FIPS indicates whether FIPS mode is enabled.
	FIPS bool `json:"fips,omitempty"`

	// KernelType indicates the type of kernel.
	KernelType string `json:"kernelType,omitempty"`
}

// MachineConfigList contains a list of MachineConfigs.
// +kubebuilder:object:root=true
type MachineConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineConfig `json:"items"`
}
