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
	// OCI configuration for the cluster autoscaler
	OCI OCIConfig `json:"oci"`

	// Autoscaling configuration
	Autoscaling AutoscalingConfig `json:"autoscaling"`

	// CAPI configuration
	CAPI CAPIConfig `json:"capi,omitempty"`

	// ClusterAutoscaler configuration
	ClusterAutoscaler ClusterAutoscalerConfig `json:"clusterAutoscaler,omitempty"`
}

// OCIConfig contains OCI-specific configuration
type OCIConfig struct {
	// TenancyID is the OCI tenancy OCID
	TenancyID string `json:"tenancyId"`

	// UserID is the OCI user OCID
	UserID string `json:"userId"`

	// Region is the OCI region
	Region string `json:"region"`

	// Fingerprint for the API key
	Fingerprint string `json:"fingerprint"`

	// PrivateKeySecretRef references a secret containing the private key
	PrivateKeySecretRef SecretRef `json:"privateKeySecretRef"`

	// CompartmentID is the OCI compartment OCID
	CompartmentID string `json:"compartmentId"`

	// ImageID is the OCID of the custom RHCOS image
	ImageID string `json:"imageId"`

	// Network configuration
	Network NetworkConfig `json:"network"`
}

// NetworkConfig contains OCI network configuration
type NetworkConfig struct {
	// VCNID is the Virtual Cloud Network OCID
	VCNID string `json:"vcnId"`

	// SubnetID is the subnet OCID for worker nodes
	SubnetID string `json:"subnetId"`

	// NetworkSecurityGroupID for worker nodes
	NetworkSecurityGroupID string `json:"networkSecurityGroupId"`

	// APIServerLoadBalancerID is the OCID of the API server load balancer
	APIServerLoadBalancerID string `json:"apiServerLoadBalancerId"`

	// ControlPlaneEndpoint is the control plane endpoint IP/hostname
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint"`
}

// AutoscalingConfig contains autoscaling configuration
type AutoscalingConfig struct {
	// minNodes is the minimum number of nodes in the autoscaling group
	// +kubebuilder:validation:Minimum=0
	MinNodes int32 `json:"minNodes,omitempty"`

	// maxNodes is the maximum number of nodes in the autoscaling group
	MaxNodes int32 `json:"maxNodes"`

	// nodeShape is the OCI compute shape for autoscaling nodes
	Shape string `json:"shape"`

	// ShapeConfig contains flexible shape configuration
	ShapeConfig *ShapeConfig `json:"shapeConfig,omitempty"`
}

// ShapeConfig contains OCI flexible shape configuration
type ShapeConfig struct {
	// CPUs is the number of OCPUs
	CPUs int32 `json:"cpus"`

	// Memory is the amount of memory in GB
	Memory int32 `json:"memory"`
}

// CAPIConfig contains Cluster API configuration
type CAPIConfig struct {
	// Namespace where CAPI resources will be created
	Namespace string `json:"namespace,omitempty"`

	// ClusterName is the name of the CAPI cluster
	ClusterName string `json:"clusterName,omitempty"`
}

// ClusterAutoscalerConfig contains cluster-autoscaler specific configuration
type ClusterAutoscalerConfig struct {
	// Image is the cluster-autoscaler image to use
	Image string `json:"image,omitempty"`

	// Resources defines resource requirements for cluster-autoscaler
	Resources *ResourceRequirements `json:"resources,omitempty"`
}

// ResourceRequirements contains resource requirements
type ResourceRequirements struct {
	// Requests describes the minimum amount of compute resources required
	Requests map[string]string `json:"requests,omitempty"`

	// Limits describes the maximum amount of compute resources allowed
	Limits map[string]string `json:"limits,omitempty"`
}

// SecretRef references a secret
type SecretRef struct {
	// Name is the name of the secret
	Name string `json:"name"`

	// Key is the key in the secret
	Key string `json:"key,omitempty"`
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
