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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IncusMachineSpec defines the desired state of IncusMachine.
type IncusMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ProviderID will be the container name in ProviderID format
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// InstanceSpec is the instance configuration
	InstanceSpec InstanceSpec `json:"instanceSpec,omitempty"`
}

// InstanceType represents the type of instance
type InstanceType string

const (
	// InstanceTypeContainer represents a container instance
	InstanceTypeContainer InstanceType = "container"
	// InstanceTypeVirtualMachine represents a virtual machine instance
	InstanceTypeVirtualMachine InstanceType = "virtual-machine"
)

type InstanceSource struct {
	// Source type
	// Example: image
	// +kubebuilder:default=image
	// +optional
	Type string `json:"type" yaml:"type"`

	// Certificate (for remote images or migration)
	// Example: X509 PEM certificate
	// +optional
	Certificate string `json:"certificate" yaml:"certificate"`

	// Image alias name (for image source)
	// Example: ubuntu/22.04
	// +optional
	Alias string `json:"alias,omitempty" yaml:"alias,omitempty"`

	// Image fingerprint (for image source)
	// Example: ed56997f7c5b48e8d78986d2467a26109be6fb9f2d92e8c7b08eb8b6cec7629a
	// +optional
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`

	// Image filters (for image source)
	// Example: {"os": "Ubuntu", "release": "jammy", "variant": "cloud"}
	// +optional
	Properties map[string]string `json:"properties,omitempty" yaml:"properties,omitempty"`

	// Remote server URL (for remote images)
	// Example: https://images.linuxcontainers.org
	// +optional
	Server string `json:"server,omitempty" yaml:"server,omitempty"`

	// Remote server secret (for remote private images)
	// Example: RANDOM-STRING
	// +optional
	Secret string `json:"secret,omitempty" yaml:"secret,omitempty"`

	// Protocol name (for remote image)
	// Example: simplestreams
	// +optional
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`

	// Base image fingerprint (for faster migration)
	// Example: ed56997f7c5b48e8d78986d2467a26109be6fb9f2d92e8c7b08eb8b6cec7629a
	// +optional
	BaseImage string `json:"baseImage,omitempty" yaml:"baseImage,omitempty"`

	// Whether to use pull or push mode (for migration)
	// Example: pull
	// +kubebuilder:default=pull
	// +optional
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`

	// Remote operation URL (for migration)
	// Example: https://1.2.3.4:8443/1.0/operations/1721ae08-b6a8-416a-9614-3f89302466e1
	// +optional
	Operation string `json:"operation,omitempty" yaml:"operation,omitempty"`

	// Map of migration websockets (for migration)
	// Example: {"criu": "RANDOM-STRING", "rsync": "RANDOM-STRING"}
	// +optional
	Websockets map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`

	// Existing instance name or snapshot (for copy)
	// Example: foo/snap0
	// +optional
	Source string `json:"source,omitempty" yaml:"source,omitempty"`

	// Whether this is a live migration (for migration)
	// Example: false
	// +optional
	Live bool `json:"live,omitempty" yaml:"live,omitempty"`

	// Whether the copy should skip the snapshots (for copy)
	// Example: false
	// +optional
	InstanceOnly bool `json:"instanceOnly,omitempty" yaml:"instanceOnly,omitempty"`

	// Whether this is refreshing an existing instance (for migration and copy)
	// Example: false
	// +optional
	Refresh bool `json:"refresh,omitempty" yaml:"refresh,omitempty"`

	// Source project name (for copy and local image)
	// Example: blah
	// +optional
	Project string `json:"project,omitempty" yaml:"project,omitempty"`

	// Whether to ignore errors when copying (e.g. for volatile files)
	// Example: false
	//
	// API extension: instance_allow_inconsistent_copy
	// +optional
	AllowInconsistent bool `json:"allowInconsistent" yaml:"allowInconsistent"`
}

type InstanceSpec struct {
	// Type (container or virtual-machine)
	// Example: container
	// +kubebuilder:default=container
	// +optional
	Type InstanceType `json:"type" yaml:"type"`

	// Source of the instance
	// +optional
	Source InstanceSource `json:"source" yaml:"source"`

	// Architecture name
	// Example: x86_64
	// +kubebuilder:default=x86_64
	// +optional
	Architecture string `json:"architecture" yaml:"architecture"`

	// Instance configuration (see doc/instances.md)
	// Example: {"security.nesting": "true"}
	// +optional
	Config map[string]string `json:"config" yaml:"config"`

	// Instance devices (see doc/instances.md)
	// Example: {"root": {"type": "disk", "pool": "default", "path": "/"}}
	// +optional
	Devices map[string]map[string]string `json:"devices" yaml:"devices"`

	// Whether the instance is ephemeral (deleted on shutdown)
	// Example: false
	// +optional
	Ephemeral bool `json:"ephemeral" yaml:"ephemeral"`

	// List of profiles applied to the instance
	// Example: ["default"]
	// +optional
	Profiles []string `json:"profiles" yaml:"profiles"`

	// If set, instance will be restored to the provided snapshot name
	// Example: snap0
	// +optional
	Restore string `json:"restore,omitempty" yaml:"restore,omitempty"`

	// Whether the instance currently has saved state on disk
	// Example: false
	// +optional
	Stateful bool `json:"stateful" yaml:"stateful"`

	// Instance description
	// Example: My test instance
	// +optional
	Description string `json:"description" yaml:"description"`
}

// IncusMachineStatus defines the observed state of IncusMachine.
type IncusMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready denotes that the machine is ready
	// +optional
	Ready bool `json:"ready"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IncusMachine is the Schema for the incusmachines API.
type IncusMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncusMachineSpec   `json:"spec,omitempty"`
	Status IncusMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IncusMachineList contains a list of IncusMachine.
type IncusMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IncusMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IncusMachine{}, &IncusMachineList{})
}
