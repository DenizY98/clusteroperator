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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EdgeClusterSpec defines the desired state of EdgeCluster
type EdgeClusterSpec struct {
	// Unique identifier for the edge cluster
	ID int `json:"id,omitempty"`
	// Identifier for the cloud provider
	ProviderID int `json:"providerId,omitempty"`
	// Kubeconfig file content for accessing the cluster
	Kubeconf string `json:"kubeconf,omitempty"`
	// Names of the image pull secrets
	ImagePullSecretNames string `json:"imagePullSecretNames,omitempty"`
	// Path to the container registry
	RegistryPath string `json:"registryPath,omitempty"`
	// Application ID for Dapr
	DaprAppId string `json:"daprAppId,omitempty"`
}

// EdgeClusterStatus defines the observed state of EdgeCluster
type EdgeClusterStatus struct {
	// ServiceName is the name of the service currently serving this EdgeCluster
	ServiceName string `json:"serviceName,omitempty"`
	// DeploymentName is the name of the Deployment currently serving this EdgeCluster
	DeploymentName string `json:"deploymentName,omitempty"`
	// Spec mirrors the current spec of the EdgeCluster
	Spec EdgeClusterSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EdgeCluster is the Schema for the edgeclusters API
type EdgeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeClusterSpec   `json:"spec,omitempty"`
	Status EdgeClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EdgeClusterList contains a list of EdgeCluster
type EdgeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeCluster{}, &EdgeClusterList{})
}
