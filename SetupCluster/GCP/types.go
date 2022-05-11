package GCP

type Cluster struct {
	Cloud struct {
		Name            string `yaml:"Name"`
		Project         string `yaml:"Project"`
		Region          string `yaml:"Region"`
		Cluster         string `yaml:"Cluster"`
		CredentialsPath string `yaml:"CredentialsPath"`
	} `yaml:"Cloud"`
	VPC struct {
		// Doesn't allow deleting existing subnets during updation
		Subnets               map[string]string `yaml:"Subnets"`
		AutoCreateSubnetworks *bool             `yaml:"AutoCreateSubnetworks"`
	} `yaml:"VPC"`
	Master struct {
		KubernetesVersion string `yaml:"KubernetesVersion"`
	} `yaml:"Master"`
	Cluster struct {
		SubnetId             string            `yaml:"SubnetId"`
		Labels               map[string]string `yaml:"Labels"`
		ServiceCIDR          string            `yaml:"ServiceCidr"`
		PrivateClusterConfig *struct {
			ControlPlaneExternalAccess bool   `yaml:"ControlPlaneExternalAccess"`
			ControlPlaneGlobalAccess   bool   `yaml:"ControlPlaneGlobalAccess"`
			ControlPlaneCidr           string `yaml:"ControlPlaneCidr"`
		} `yaml:"PrivateClusterConfig"`
		VPA               bool `yaml:"Vpa"`
		VPCNativeRouting  bool `yaml:"VPCNativeRouting"`
		CloudLogging      bool `yaml:"CloudLogging"`
		CloudMonitoring   bool `yaml:"CloudMonitoring"`
		Prometheus        bool `yaml:"Prometheus"`
		NetworkPolicy     bool `yaml:"NetworkPolicy"`
		HttpLoadBalancer  bool `yaml:"HttpLoadBalancer"`
		ShieldedNodes     bool `yaml:"ShieldedNodes"`
		ManagedPrometheus bool `yaml:"ManagedPrometheus"`
		WorkloadIdentity  bool `yaml:"WorkloadIdentity"`
	} `yaml:"Cluster"`
	Nodes []struct {
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
		ScalingConfig struct {
			DesiredSize int32 `yaml:"DesiredSize"`
			MinSize     int32 `yaml:"MinSize"`
			MaxSize     int32 `yaml:"MaxSize"`
		} `yaml:"ScalingConfig"`
	} `yaml:"Nodes"`
}
