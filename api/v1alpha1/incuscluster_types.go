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

const (
	// ClusterFinalizer allows IncusClusterReconciler to clean up resources associated with IncusCluster before
	// removing it from the API server.
	ClusterFinalizer = "incuscluster.infrastructure.cluster.x-k8s.io"
	// MachineFinalizer allows IncusMachineReconciler to clean up resources associated with IncusMachine before
	// removing it from the API server.
	MachineFinalizer = "incusmachine.infrastructure.cluster.x-k8s.io"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IncusClusterSpec defines the desired state of IncusCluster.
type IncusClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint APIEndpoint `json:"controlPlaneEndpoint"`

	// Incus is the configuration for the Incus API.
	Incus Incus `json:"incus"`
}

type Incus struct {
	// Endpoint is the API endpoint to reach the Incus API.
	Endpoint string `json:"endpoint"`

	// secretRef is a reference to the secret containing the credentials to access the Incus API.
	SecretRef SecretReference `json:"secretRef"`

	// Project is the project to use when interacting with the Incus API.
	Project string `json:"project"`
}

type SecretReference struct {
	// Name is the name of the secret.
	Name string `json:"name"`
}

// APIEndpoint represents a reachable Kubernetes API endpoint.
type APIEndpoint struct {
	// Host is the hostname on which the API server is serving.
	Host string `json:"host"`

	// Port is the port on which the API server is serving.
	// Defaults to 6443 if not set.
	Port int `json:"port"`
}

// IncusClusterStatus defines the observed state of IncusCluster.
type IncusClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Ready denotes that the in-memory cluster (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IncusCluster is the Schema for the incusclusters API.
type IncusCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncusClusterSpec   `json:"spec,omitempty"`
	Status IncusClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IncusClusterList contains a list of IncusCluster.
type IncusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IncusCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IncusCluster{}, &IncusClusterList{})
}
