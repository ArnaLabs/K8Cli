package SetupCluster

import (
	"fmt"
	ekssetup "github.com/ArnaLabs/K8Cli/SetupCluster/EKS"
	"gopkg.in/yaml.v2"
	"io/ioutil"
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

func CheckCluster(f string, context string) {

	////Reading inputs from yaml

	file := f
	var cloud CldDetails

	fileConfigYml, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
	}

	err = yaml.Unmarshal([]byte(fileConfigYml), &cloud)
	//fmt.Println(cloud)
	if err != nil {
		panic(err)
	}

	if cloud.Cloud.Name == "AWS" {
		fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
		fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
		fmt.Printf("Cluster: %#v\n", cloud.Cloud.Cluster)
		fmt.Printf("Bucket: %#v\n", cloud.Cloud.Bucket)
		//fmt.Println("Setting up EKS Cluster ........")
		//Passing cluster file
		if context == cloud.Cloud.Cluster {
			ekssetup.ReadEKSYaml([]byte(fileConfigYml))
		} else {
			fmt.Println("ClusterName doesn't match with Context name. Please validate cluster yml")
		}
	}

	//End EKS Cluster elements session values
	//if cloud.Cloud.Name == "Azure" {
	//	fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
	//	fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
	//	fmt.Println("Setting up AKS Cluster")
	//}
}
