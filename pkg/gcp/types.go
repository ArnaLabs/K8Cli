package gcp

type Cluster struct {
	Cloud struct {
		Name            string `yaml:"Name"`
		Project         string `yaml:"Project"`
		Region          string `yaml:"Region"`
		Cluster         string `yaml:"Cluster"`
		CredentialsPath string `yaml:"CredentialsPath"`
	} `yaml:"Cloud"`
	VPC struct {
		VpcBlock              *string           `yaml:"VpcBlock"`
		Subnets               map[string]string `yaml:"Subnets"`
		AutoCreateSubnetworks *bool             `yaml:"AutoCreateSubnetworks"`
	} `yaml:"VPC"`
	Master struct {
		KubernetesVersion string `yaml:"KubernetesVersion"`
	} `yaml:"Master"`
	Cluster struct {
		Labels       map[string]string `yaml:"labels"`
		ServiceCIDR  string            `yaml:"serviceCidr"`
		PrivateNodes bool              `yaml:"PrivateNodes"`
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
