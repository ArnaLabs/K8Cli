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
	Nodes []struct {
		NodeGroupName string            `yaml:"NodegroupName"`
		MachineType   string            `yaml:"MachineType"`
		Tags          []string          `yaml:"Tags"`
		Labels        map[string]string `yaml:"Labels"`
	} `yaml:"Nodes"`
}
