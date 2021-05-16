package SetupCluster

import (
	"fmt"
	"github.com/K8-Cloud/k8-cloud/SetupCluster/EKS"
	"gopkg.in/yaml.v2"
)

type CldDetails struct {
	Cloud struct {
		Name    string `yaml:"Name"`
		Region  string `yaml:"Region"`
		Cluster string `yaml:"Cluster"`
		Bucket  string `yaml:"Bucket"`
	} `yaml:"Cloud"`
}

//Setup AKS or EKS Cluster

func CheckCluster(f []byte) {

	////Reading inputs from yaml

	file := f
	var cloud CldDetails
	//m := make(map[interface{}]interface{})
	err := yaml.Unmarshal([]byte(file), &cloud)
	fmt.Println(cloud)
	if err != nil {
		panic(err)
	}

	if cloud.Cloud.Name == "AWS" {
		fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
		fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
		fmt.Printf("Cluster: %#v\n", cloud.Cloud.Cluster)
		fmt.Printf("Bucket: %#v\n", cloud.Cloud.Bucket)
		fmt.Println("Setting up EKS Cluster ........")
		//Passing cluster file
		ekssetup.ReadEKSYaml(file)
	}


	//End EKS Cluster elements session values
	//if cloud.Cloud.Name == "Azure" {
	//	fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
	//	fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
	//	fmt.Println("Setting up AKS Cluster")
	//}
}
