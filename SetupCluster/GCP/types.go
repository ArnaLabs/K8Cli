package GCP

type NodePoolConfig struct {
	NodeGroupName string            `yaml:"NodegroupName"`
	MachineType   string            `yaml:"MachineType"`
	DiskSize      int32             `yaml:"DiskSize"`
	Tags          []string          `yaml:"Tags"`
	Labels        map[string]string `yaml:"Labels"`
	SpotInstance  bool              `yaml:"SpotInstance"`
	Taints        []struct {
		Effect string `yaml:"Effect"`
		Key    string `yaml:"Key"`
		Value  string `yaml:"Value"`
	} `yaml:"Taints"`
	// Can be updated
	ScalingConfig struct {
		DesiredSize int32 `yaml:"DesiredSize"`
		MinSize     int32 `yaml:"MinSize"`
		MaxSize     int32 `yaml:"MaxSize"`
	} `yaml:"ScalingConfig"`
}

type Cluster struct {
	Cloud struct {
		Name    string `yaml:"Name"`
		Project string `yaml:"Project"`
		Region  string `yaml:"Region"`
		Cluster string `yaml:"Cluster"`
		// Optional, will use currently logged in gcloud cli creds if CredentialsPath is not provided
		CredentialsPath string `yaml:"CredentialsPath"`
	} `yaml:"Cloud"`
	VPC struct {
		// Doesn't allow deleting existing subnets
		Subnets               map[string]string `yaml:"Subnets"`
		AutoCreateSubnetworks *bool             `yaml:"AutoCreateSubnetworks"`
		// Provide name of an existing subnet if you don't want a new subnet to be created
		ExistingVPC string `yaml:"ExistingVPC"`
	} `yaml:"VPC"`
	Master struct {
		// Can be upgraded
		KubernetesVersion string `yaml:"KubernetesVersion"`
	} `yaml:"Master"`
	Cluster struct {
		// Name of the subnet as per the list given at vpc section above
		SubnetId string `yaml:"SubnetId"`
		// Can be updated
		Labels map[string]string `yaml:"Labels"`
		// CIDR for k8s services. Doesn't have to be in the VPC.
		ServiceCIDR          string `yaml:"ServiceCidr"`
		PrivateClusterConfig *struct {
			ControlPlaneExternalAccess bool   `yaml:"ControlPlaneExternalAccess"`
			ControlPlaneGlobalAccess   bool   `yaml:"ControlPlaneGlobalAccess"`
			ControlPlaneCidr           string `yaml:"ControlPlaneCidr"`
		} `yaml:"PrivateClusterConfig"`
		// Vertical Pod Autoscaler
		VPA              bool `yaml:"Vpa"`
		VPCNativeRouting bool `yaml:"VPCNativeRouting"`
		// Enable cloud logging
		CloudLogging bool `yaml:"CloudLogging"`
		// Enable cloud monitoring
		CloudMonitoring bool `yaml:"CloudMonitoring"`
		Prometheus      bool `yaml:"Prometheus"`
		// Enable NetworkPolicy using calico
		NetworkPolicy bool `yaml:"NetworkPolicy"`
		// Can be updated
		HttpLoadBalancer  bool `yaml:"HttpLoadBalancer"`
		ShieldedNodes     bool `yaml:"ShieldedNodes"`
		ManagedPrometheus bool `yaml:"ManagedPrometheus"`
		WorkloadIdentity  bool `yaml:"WorkloadIdentity"`
	} `yaml:"Cluster"`
	Nodes []NodePoolConfig `yaml:"Nodes"`
}
