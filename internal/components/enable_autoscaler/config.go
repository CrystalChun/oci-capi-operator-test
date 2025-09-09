package enableautoscaler

import (
	"context"
	"fmt"

	ocicapioperatorv1alpha1 "github.com/openshift/oci-capi-operator/api/v1alpha1"
	"github.com/openshift/oci-capi-operator/internal/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	ClusterConfig     ClusterConfig
	NetworkConfig     NetworkConfig
	AutoScalingConfig AutoScalingConfig
}

type AutoScalingConfig struct {
	CPUs     int32  `envconfig:"AUTOSCALER_CPUS" default:"2"`
	Memory   int32  `envconfig:"AUTOSCALER_MEMORY" default:"4"`
	MinNodes int32  `envconfig:"AUTOSCALER_MIN_NODES" default:"1"`
	MaxNodes int32  `envconfig:"AUTOSCALER_MAX_NODES" default:"3"`
	Shape    string `envconfig:"AUTOSCALER_SHAPE" default:"oc3"`
	ImageID  string `envconfig:"IMAGE_ID" default:""`
}

type ClusterConfig struct {
	CompartmentID string `envconfig:"COMPARTMENT_ID" default:""`
}

type NetworkConfig struct {
	VCNID                   string `envconfig:"VCN_ID" default:""`
	SubnetID                string `envconfig:"SUBNET_ID" default:""`
	NetworkSecurityGroupID  string `envconfig:"NETWORK_SECURITY_GROUP_ID" default:""`
	ControlPlaneEndpoint    string `envconfig:"CONTROL_PLANE_ENDPOINT" default:""`
	APIServerLoadBalancerID string `envconfig:"API_SERVER_LOAD_BALANCER_ID" default:""`
	ClusterNetworkCIDRBlock string `envconfig:"CLUSTER_NETWORK_CIDR_BLOCK" default:""`
	ServiceNetworkCIDRBlock string `envconfig:"SERVICE_NETWORK_CIDR_BLOCK" default:""`
}

func SetAutoScalingConfig(ctx context.Context, client client.Client, instance *ocicapioperatorv1alpha1.OCIClusterAutoscaler, config Config) (Config, error) {
	if instance.Spec.Autoscaling.ShapeConfig != nil {
		if instance.Spec.Autoscaling.ShapeConfig.CPUs != 0 {
			config.AutoScalingConfig.CPUs = instance.Spec.Autoscaling.ShapeConfig.CPUs
		}
		if instance.Spec.Autoscaling.ShapeConfig.Memory != 0 {
			config.AutoScalingConfig.Memory = instance.Spec.Autoscaling.ShapeConfig.Memory
		}
	}
	if instance.Spec.Autoscaling.MinNodes != 0 {
		config.AutoScalingConfig.MinNodes = instance.Spec.Autoscaling.MinNodes
	}
	if instance.Spec.Autoscaling.MaxNodes != 0 {
		config.AutoScalingConfig.MaxNodes = instance.Spec.Autoscaling.MaxNodes
	}

	if instance.Spec.Autoscaling.Shape != "" {
		config.AutoScalingConfig.Shape = instance.Spec.Autoscaling.Shape
	}
	if instance.Spec.Autoscaling.ImageID != "" {
		config.AutoScalingConfig.ImageID = instance.Spec.Autoscaling.ImageID
	}
	if config, err := SetNetworkConfig(ctx, client, config); err != nil {
		return config, fmt.Errorf("failed to set network config: %w", err)
	}

	return config, nil
}

// SetNetworkConfig finds the network CIDRs in the cluster if it is not set in the config
// then sets them in the config
func SetNetworkConfig(ctx context.Context, client client.Client, config Config) (Config, error) {
	if config.NetworkConfig.ClusterNetworkCIDRBlock == "" {
		clusterNetworkCIDRBlock, err := utils.GetClusterNetworkCIDRBlock(ctx, client)
		if err != nil {
			return config, fmt.Errorf("failed to get cluster network CIDR block: %w", err)
		}
		config.NetworkConfig.ClusterNetworkCIDRBlock = clusterNetworkCIDRBlock
	}
	if config.NetworkConfig.ServiceNetworkCIDRBlock == "" {
		serviceNetworkCIDRBlock, err := utils.GetServiceNetworkCIDRBlock(ctx, client)
		if err != nil {
			return config, fmt.Errorf("failed to get service network CIDR block: %w", err)
		}
		config.NetworkConfig.ServiceNetworkCIDRBlock = serviceNetworkCIDRBlock
	}
	return config, nil
}
