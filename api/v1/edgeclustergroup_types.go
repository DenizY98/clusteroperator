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

// EdgeClusterGroupSpec defines the desired state of EdgeClusterGroup
type EdgeClusterGroupSpec struct {
	// DaprAppId is the identifier for the group of edge clusters
	DaprAppId string `json:"daprAppId"`
	// Clusters is the list of EdgeClusters that are part of this group
	Clusters []EdgeClusterSpec `json:"clusters"`
}

// EdgeClusterGroupStatus defines the observed state of EdgeClusterGroup
type EdgeClusterGroupStatus struct {
	// TotalClusters indicates the total number of EdgeClusters in the group
	TotalClusters int `json:"totalClusters,omitempty"`
	// ReadyClusters indicates the number of EdgeClusters that are ready
	ReadyClusters int `json:"readyClusters,omitempty"`
	// UnreadyClusters indicates the number of EdgeClusters that are not ready
	UnreadyClusters int `json:"unreadyClusters,omitempty"`
	// GroupStatus indicates the overall status of the EdgeClusterGroup
	GroupStatus string `json:"groupStatus,omitempty"`
	// LastUpdated represents the last time the status was updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=ecg
// +kubebuilder:subresource:status

// EdgeClusterGroup is the Schema for the edgeclustergroups API
type EdgeClusterGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EdgeClusterGroupSpec   `json:"spec,omitempty"`
	Status EdgeClusterGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EdgeClusterGroupList contains a list of EdgeClusterGroup
type EdgeClusterGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EdgeClusterGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EdgeClusterGroup{}, &EdgeClusterGroupList{})
}
