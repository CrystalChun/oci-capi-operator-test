package enableautoscaler

type Config struct {
	ClusterConfig     ClusterConfig
	NetworkConfig     NetworkConfig
	AutoScalingConfig AutoScalingConfig
}

type AutoScalingConfig struct {
	CPUs     int32  `envconfig:"AUTOSCALER_CPUs" default:"2"`
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
