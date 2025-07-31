package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OCICAPIClusterSpec defines the desired state of OCICAPICluster
type OCICAPIClusterSpec struct {
	// OCI Configuration
	// +required
	OCI OCIConfig `json:"oci"`

	// Autoscaling Configuration
	// +required
	Autoscaling AutoscalingConfig `json:"autoscaling"`

	// Network Configuration
	// +required
	Network NetworkConfig `json:"network"`

	// Image Configuration
	// +required
	Image ImageConfig `json:"image"`
}

// OCIConfig contains OCI-specific configuration
type OCIConfig struct {
	// CompartmentID is the OCID of the compartment where resources will be created
	// +required
	CompartmentID string `json:"compartmentId"`

	// Region is the OCI region where resources will be created
	// +required
	Region string `json:"region"`

	// Credentials contains OCI credentials configuration
	// +required
	Credentials CredentialsConfig `json:"credentials"`
}

// CredentialsConfig contains OCI credentials
type CredentialsConfig struct {
	// TenancyID is the OCID of the tenancy
	// +required
	TenancyID string `json:"tenancyId"`

	// UserID is the OCID of the user
	// +required
	UserID string `json:"userId"`

	// PrivateKeySecretRef references the secret containing the private key
	// +required
	PrivateKeySecretRef SecretReference `json:"privateKeySecretRef"`

	// Fingerprint is the fingerprint of the public key
	// +required
	Fingerprint string `json:"fingerprint"`

	// Passphrase is the optional passphrase for the private key
	// +optional
	Passphrase string `json:"passphrase,omitempty"`
}

// SecretReference contains details about a secret reference
type SecretReference struct {
	// Name is the name of the secret
	// +required
	Name string `json:"name"`

	// Namespace is the namespace of the secret
	// +required
	Namespace string `json:"namespace"`

	// Key is the key in the secret data
	// +required
	Key string `json:"key"`
}

// AutoscalingConfig contains autoscaling configuration
type AutoscalingConfig struct {
	// MinNodes is the minimum number of nodes in the autoscaling group
	// +required
	MinNodes int32 `json:"minNodes"`

	// MaxNodes is the maximum number of nodes in the autoscaling group
	// +required
	MaxNodes int32 `json:"maxNodes"`

	// NodeShape is the OCI compute shape for worker nodes
	// +required
	NodeShape string `json:"nodeShape"`

	// ShapeConfig contains the configuration for the node shape
	// +required
	ShapeConfig ShapeConfig `json:"shapeConfig"`
}

// ShapeConfig contains the configuration for the node shape
type ShapeConfig struct {
	// OCPUs is the number of OCPUs for the shape
	// +required
	OCPUs int32 `json:"ocpus"`

	// MemoryInGBs is the amount of memory in GBs
	// +required
	MemoryInGBs int32 `json:"memoryInGBs"`
}

// NetworkConfig contains network configuration
type NetworkConfig struct {
	// VCNID is the OCID of the VCN
	// +required
	VCNID string `json:"vcnId"`

	// SubnetID is the OCID of the subnet for worker nodes
	// +required
	SubnetID string `json:"subnetId"`

	// NetworkSecurityGroupID is the OCID of the network security group for worker nodes
	// +required
	NetworkSecurityGroupID string `json:"networkSecurityGroupId"`

	// APIServerLoadBalancerID is the OCID of the API server load balancer
	// +required
	APIServerLoadBalancerID string `json:"apiServerLoadBalancerId"`

	// ControlPlaneEndpoint is the endpoint for the control plane
	// +required
	ControlPlaneEndpoint string `json:"controlPlaneEndpoint"`
}

// ImageConfig contains image configuration
type ImageConfig struct {
	// ImageID is the OCID of the RHCOS image
	// +required
	ImageID string `json:"imageId"`
}

// OCICAPIClusterStatus defines the observed state of OCICAPICluster
type OCICAPIClusterStatus struct {
	// Conditions represent the latest available observations of the cluster's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current phase of cluster actuation
	// +optional
	Phase string `json:"phase,omitempty"`

	// ObservedGeneration is the last observed generation of the OCICAPICluster
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// OCICAPICluster is the Schema for the ocicapiclusters API
type OCICAPICluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OCICAPIClusterSpec   `json:"spec,omitempty"`
	Status OCICAPIClusterStatus `json:"status,omitempty"`
}

func init() {
	SchemeBuilder.Register(&OCICAPICluster{}, &OCICAPIClusterList{})
}

//+kubebuilder:object:root=true

// OCICAPIClusterList contains a list of OCICAPICluster
type OCICAPIClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OCICAPICluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OCICAPICluster{}, &OCICAPIClusterList{})
}
