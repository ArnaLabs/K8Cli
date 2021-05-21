package main

import (
	"flag"
	"fmt"
	"github.com/ArnaLabs/K8Cli/SetupCluster"
	_ "github.com/ArnaLabs/K8Cli/SetupCluster/EKS"
	"github.com/ArnaLabs/K8Cli/manageCluster"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
)

func main() {
	var masterurl, kubeconfigfile, clusterfile, helmconfig string
	var InitialConfigVals InitialConfigVals
	var config HelmConfig
	var operation, context, name string

	flag.StringVar(&operation, "operation", "cluster", "Provide whether operation needed to be performed - Cluster/Addons")
	flag.StringVar(&context, "context", "minikube", "Provide kubernetes context for addon")
	flag.StringVar(&name, "name", "backup", "backup name")
	flag.StringVar(&kubeconfigfile, "kube-config", "", "Provide path to kubeconfig")
	version := flag.Bool("version", false, "display version")

	flag.Parse()

	if *version {
		fmt.Print("K8Cli version: 1.0.0\n")
		os.Exit(0)
	}

	var manageOperation = StrSlice{"cluster", "addons", "resource-all", "namespace", "storage", "resourcequota", "defaultquota", "serviceaccount"}

	if manageOperation.Has(operation) {

		filePath := "K8Cli/" + context + "/config.yml"
		fmt.Println(filePath)

		fileConfigYml, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Println(err)
		}

		err = yaml.Unmarshal([]byte(fileConfigYml), &InitialConfigVals)
		if err != nil {
			panic(err)
		}

		if operation == "cluster" {

			clusterfile = InitialConfigVals.ClusterDetails.ClusterYaml
			fmt.Println("Path to cluster yml: ", clusterfile)

		} else {

			clusterfile = InitialConfigVals.ClusterDetails.ClusterYaml
			fmt.Println("Path to cluster yml: ", clusterfile)

			kubeconfigfile = InitialConfigVals.ClusterDetails.KubeConfig
			fmt.Println("Path to kubeconfig: ", kubeconfigfile)

			if kubeconfigfile == "" {
				dirname, err := os.UserHomeDir()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(dirname)

				kubeconfigfile = dirname + "/.kube/config"
			}
			masterurl, err = getClusterEndpoint(context, kubeconfigfile)
			if err != nil {
				masterurl = " "
			}

			helmconfig = InitialConfigVals.ClusterDetails.Addons
			err = yaml.Unmarshal([]byte(helmconfig), &config)

		}

	}

	if operation == "addons" {

		helmInit(context)

		helmAddRepositories(config)
		fmt.Print(config)
		helmInstallReleases(config, context)

	} else if operation == "cluster" {

		SetupCluster.CheckCluster(clusterfile, context)

	} else if operation == "take_backup" {
		takeBackup(name, context)
		fmt.Print("Work In Progress\n")

	} else if operation == "resource-all" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)

		fmt.Println("Executing Create or Update StorageClasses")
		manageCluster.CreateorUpdateStorageClass(InitialConfigVals.ClusterDetails.StorageClassFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

		fmt.Println("Executing Create or Update DefaultQuotas")
		manageCluster.CreateorUpdateDefaultQuota(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

		fmt.Println("Executing Create or Update ResourceQuotas")
		manageCluster.CreateorUpdateResourceQuota(InitialConfigVals.ClusterDetails.ResourceQuotaFile, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

		fmt.Println("Executing Create or Update NameSpaceUsers")
		manageCluster.CreateorUpdateNameSpaceUser(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "namespace" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "storage" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)
		fmt.Println("Executing Create or Update StorageClasses")
		manageCluster.CreateorUpdateStorageClass(InitialConfigVals.ClusterDetails.StorageClassFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "resourcequota" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update DefaultQuotas")
		manageCluster.CreateorUpdateDefaultQuota(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update ResourceQuotas")
		manageCluster.CreateorUpdateResourceQuota(InitialConfigVals.ClusterDetails.ResourceQuotaFile, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "defaultquota" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update DefaultQuotas")
		manageCluster.CreateorUpdateDefaultQuota(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "serviceaccount" {

		connection := setupK8sConnection(InitialConfigVals, masterurl)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update NameSpaceUsers")
		manageCluster.CreateorUpdateNameSpaceUser(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "init" {

		if kubeconfigfile == "" {
			dirname, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(dirname)

			kubeconfigfile = dirname + "/.kube/config"
		}

		fmt.Println("Initializing K8Cli")
		fmt.Printf("ClusterName: %v\n", context)
		fmt.Printf("Kubeconfigfile: %v\n", kubeconfigfile)

		//manageCluster.Init(context, kubeconfigfile)
		Init(context, kubeconfigfile)

	} else if operation == "" {

		fmt.Printf("MasterUrl: %v\n", masterurl)
		fmt.Printf("KubeConfig: %v\n", InitialConfigVals.ClusterDetails.KubeConfig)
		fmt.Printf("MasterKey: %v\n", InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Printf("Configs: %v\n", InitialConfigVals.ClusterDetails.Configs)
		fmt.Printf("StorageClasses.yaml: %v\n", InitialConfigVals.ClusterDetails.StorageClassFile)
		fmt.Printf("Namepaces.yaml: %v\n", InitialConfigVals.ClusterDetails.NameSpaceFile)
		fmt.Printf("ResourceQuotas.yaml: %v\n", InitialConfigVals.ClusterDetails.ResourceQuotaFile)
		fmt.Println("Provide Valid input operation")
	} else {
		fmt.Print("Operation Not Supported")
	}

	deleteDir("templates")

}

func setupK8sConnection(InitialConfigVals InitialConfigVals, masterurl string) *kubernetes.Clientset {
	fmt.Println("Setting up Connection")
	fmt.Println(InitialConfigVals.ClusterDetails)
	fmt.Printf("MasterUrl: %v\n", masterurl)
	fmt.Printf("KubeConfig: %v\n", InitialConfigVals.ClusterDetails.KubeConfig)
	fmt.Printf("ClusterName: %v\n", InitialConfigVals.ClusterDetails.ClusterName)
	fmt.Printf("MasterKey: %v\n", InitialConfigVals.ClusterDetails.MasterKey)
	fmt.Printf("Configs: %v\n", InitialConfigVals.ClusterDetails.Configs)
	fmt.Printf("StorageClasses.yaml: %v\n", InitialConfigVals.ClusterDetails.StorageClassFile)
	fmt.Printf("Namepaces.yaml: %v\n", InitialConfigVals.ClusterDetails.NameSpaceFile)
	fmt.Printf("ResourceQuotas.yaml: %v\n", InitialConfigVals.ClusterDetails.ResourceQuotaFile)

	connection, _ := manageCluster.SetupConnection(masterurl, InitialConfigVals.ClusterDetails.KubeConfig)

	return connection
}
