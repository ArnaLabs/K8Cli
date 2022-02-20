package SetupCluster

import (
	"bufio"
	"fmt"
	ekssetup "github.com/ArnaLabs/K8Cli/SetupCluster/EKS"
	"github.com/ArnaLabs/K8Cli/pkg/gcp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

type CldDetails struct {
	Cloud struct {
		Name    string `yaml:"Name"`
		Region  string `yaml:"Region"`
		Cluster string `yaml:"Cluster"`
		Bucket  string `yaml:"Bucket"`
		Profile string `yaml:"Profile"`
	} `yaml:"Cloud"`
}

//Setup AKS or EKS Cluster

func CheckCluster(sf string, f string, context string, clustertype string, clustergreenfile string) {

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

	filegreenConfigYml, err := ioutil.ReadFile(clustergreenfile)
	if os.IsNotExist(err) {
		fmt.Printf("Green cluster config doesn't exist, ignoring\n")
	} else if err != nil {
		fmt.Printf("Unable to open green config file, err : %v", err)
	}

	//err = yaml.Unmarshal([]byte(filegreenConfigYml), &cloud)
	//fmt.Println(cloud)
	//if err != nil {
	//	panic(err)
	//}

	if cloud.Cloud.Name == "AWS" {
		fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
		fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
		fmt.Printf("Cluster: %#v\n", cloud.Cloud.Cluster)
		fmt.Printf("Bucket: %#v\n", cloud.Cloud.Bucket)
		//fmt.Println("Setting up EKS Cluster ........")
		//Passing cluster file
		if context == cloud.Cloud.Cluster {
			ekssetup.ReadEKSYaml([]byte(fileConfigYml), sf, clustertype, filegreenConfigYml)
		} else {
			fmt.Println("ClusterName doesn't match with Context name. Please validate cluster yml")
		}
	}

	if cloud.Cloud.Name == "GCP" {
		fmt.Printf("\nSetting up GCP Cluster\n")

		if err := gcp.ApplyCluster(fileConfigYml); err != nil {
			fmt.Printf("Unable to apply gcp cluster, err : %v", err)
			// Exit and return error code 1
			os.Exit(1)
		}

	}

	//End EKS Cluster elements session values
	//if cloud.Cloud.Name == "Azure" {
	//	fmt.Printf("Cloud: %#v\n", cloud.Cloud.Name)
	//	fmt.Printf("Region: %#v\n", cloud.Cloud.Region)
	//	fmt.Println("Setting up AKS Cluster")
	//}
}

func failOnError(cmd *exec.Cmd) {
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		fmt.Printf(scanner.Text())
		os.Exit(1)
	}
}

func UpdateKubeConfig(f string, context string) {
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
		if cloud.Cloud.Profile != "" {
			updateKubeConfigByProfile(cloud.Cloud.Profile, context)
		}
	}

}

func updateKubeConfigByProfile(profile string, context string) {
	os.Setenv("AWS_PROFILE", profile)
	cmd := exec.Command("aws", "eks", "update-kubeconfig", "--name", context, "--alias", context)
	failOnError(cmd)
}

func updateKubeConfigByKeys(AccessKey string, SecretKay string, Region string, context string) {
	os.Setenv("AWS_ACCESS_KEY_ID", AccessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", SecretKay)
	os.Setenv("AWS_DEFAULT_REGION", Region)
	cmd := exec.Command("aws", "eks", "update-kubeconfig", "--name", context, "--alias", context)
	failOnError(cmd)
}
