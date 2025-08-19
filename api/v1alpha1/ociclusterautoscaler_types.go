/*
Copyright 2025.

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

// OCIClusterAutoscalerSpec defines the desired state of OCIClusterAutoscaler
type OCIClusterAutoscalerSpec struct {
	// Autoscaling configuration
	Autoscaling AutoscalingConfig `json:"autoscaling"`

	// CAPI deployment configuration
	CAPI CAPIConfig `json:"capi,omitempty"`

	// ClusterAutoscaler deployment configuration
	ClusterAutoscaler ClusterAutoscalerConfig `json:"clusterAutoscaler,omitempty"`
}

// AutoscalingConfig contains optional autoscaling configuration
type AutoscalingConfig struct {
	// minNodes is the minimum number of nodes in the autoscaling group
	// +kubebuilder:validation:Minimum=0
	MinNodes int32 `json:"minNodes,omitempty"`

	// maxNodes is the maximum number of nodes in the autoscaling group
	MaxNodes int32 `json:"maxNodes,omitempty"`

	// nodeShape is the OCI compute shape for autoscaling nodes
	Shape string `json:"shape,omitempty"`

	// ShapeConfig contains flexible shape configuration
	ShapeConfig *ShapeConfig `json:"shapeConfig,omitempty"`

	// ImageID is the OCID of the custom RHCOS image for deploying new nodes during autoscaling
	ImageID string `json:"imageId,omitempty"`
}

// ShapeConfig contains OCI flexible shape configuration
type ShapeConfig struct {
	// CPUs is the number of OCPUs
	CPUs int32 `json:"cpus,omitempty"`

	// Memory is the amount of memory in GB
	Memory int32 `json:"memory,omitempty"`
}

// CAPIConfig contains Cluster API configuration
type CAPIConfig struct {
	// Namespace where CAPI resources will be created
	Namespace string `json:"namespace,omitempty"`

	// ClusterName is the name of the CAPI cluster
	ClusterName string `json:"clusterName,omitempty"`
}

// ClusterAutoscalerConfig is the configuration for the deployment of the cluster-autoscaler
type ClusterAutoscalerConfig struct {
	// RepositoryURL is the URL for the helm chart of the cluster-autoscaler
	RepositoryURL string `json:"repositoryURL,omitempty"`

	// Name is the name of the cluster-autoscaler deployment
	Name string `json:"name,omitempty"`

	// Namespace is the namespace where the cluster-autoscaler deployment will be installed
	Namespace string `json:"namespace,omitempty"`

	// ServiceAccountName is the name of the service account the cluster-autoscaler deployment will use
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// CloudProvider is the cloud provider to use for the helm chart
	CloudProvider string `json:"cloudProvider,omitempty"`

	// CreateRBAC is whether or not to create the RBAC resources from the helm chart
	// for the cluster-autoscaler
	CreateRBAC bool `json:"createRBAC,omitempty"`

	// CreateServiceAccount is whether or not to create the service account
	// from the helm chart for the cluster-autoscaler
	CreateServiceAccount bool `json:"createServiceAccount,omitempty"`

	// Version is the helm chart version of the cluster-autoscaler to install
	Version string `json:"version,omitempty"`
}

// OCIClusterAutoscalerStatus defines the observed state of OCIClusterAutoscaler
type OCIClusterAutoscalerStatus struct {
	// Conditions represent the latest available observations of the autoscaler's current state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of the autoscaler
	Phase string `json:"phase,omitempty"`

	// CAPIInstalled indicates whether CAPI components are installed
	CAPIInstalled bool `json:"capiInstalled,omitempty"`

	// ClusterAutoscalerDeployed indicates whether cluster-autoscaler is deployed
	ClusterAutoscalerDeployed bool `json:"clusterAutoscalerDeployed,omitempty"`

	// ObservedGeneration is the last generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OCIClusterAutoscaler is the Schema for the ociclusterautoscalers API
type OCIClusterAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCIClusterAutoscalerSpec   `json:"spec,omitempty"`
	Status OCIClusterAutoscalerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OCIClusterAutoscalerList contains a list of OCIClusterAutoscaler
type OCIClusterAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCIClusterAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OCIClusterAutoscaler{}, &OCIClusterAutoscalerList{})
}
