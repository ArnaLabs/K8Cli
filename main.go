package main

import (
	"flag"
	"fmt"
	"github.com/K8-Cloud/k8-cloud/SetupCluster"
	"github.com/K8-Cloud/k8-cloud/manageCluster"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"log"
	"os"
	"strings"
)

func setupK8sConnection(InitialConfigVals InitialConfigVals) *kubernetes.Clientset {
	fmt.Println("Setting up Connection")
	fmt.Println(InitialConfigVals.ClusterDetails)
	fmt.Printf("MasterUrl: %v\n", InitialConfigVals.ClusterDetails.MasterUrl)
	fmt.Printf("KubeConfig: %v\n", InitialConfigVals.ClusterDetails.KubeConfig)
	connection, _ := manageCluster.SetupConnection(InitialConfigVals.ClusterDetails.MasterUrl, InitialConfigVals.ClusterDetails.KubeConfig)
	fmt.Printf("ClusterName: %v\n", InitialConfigVals.ClusterDetails.ClusterName)
	fmt.Printf("MasterKey: %v\n", InitialConfigVals.ClusterDetails.MasterKey)
	fmt.Printf("Configs: %v\n", InitialConfigVals.ClusterDetails.Configs)
	fmt.Printf("StorageClasses.yaml: %v\n", InitialConfigVals.ClusterDetails.StorageClassFile)
	fmt.Printf("Namepaces.yaml: %v\n", InitialConfigVals.ClusterDetails.NameSpaceFile)
	fmt.Printf("ResourceQuotas.yaml: %v\n", InitialConfigVals.ClusterDetails.ResourceQuotaFile)

	return connection
}

func main() {
	var config HelmConfig
	var masterurl, kubeconfigfile string
	var InitialConfigVals InitialConfigVals
	var operation, configFile, context, name string
	//var version bool

	//flag.StringVar(&operation, "o", "all", "Provide the operation that needs to be performed, valid inputs - namespace, storage, resourcequota, defaultquota, serviceaccount")
	flag.StringVar(&operation, "operation", "cluster", "Provide whether operation needed to be performed - Cluster/Addons")
	flag.StringVar(&configFile, "config", "cf-fmt.yaml", "Provide path to Config yaml")
	flag.StringVar(&context, "context", "minikube", "Provide kubernetes context for addon")
	flag.StringVar(&name, "name", "backup", "backup name")
	flag.StringVar(&kubeconfigfile, "kube-config", "", "Provide path to kubeconfig")
	version := flag.Bool("version", false, "display version")
	//flag.StringVar(&clustername, "cluster-name", "dev-cluster", "Provide cluster name")
	//flag.StringVar(&masterurl, "u", "https://localhost:6443", "Provide master url")
	flag.Parse()

	if *version {
		fmt.Print("k8-cloud version: 1.0.0\n")
		os.Exit(0)
	}

	if operation == "addons" {
		yamlFile, err := ioutil.ReadFile(configFile)

		makeDir("templates")

		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}

		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			panic(err)
		}
		helmInit(context)
		helmAddRepositories(config)
		fmt.Print(config)
		helmInstallReleases(config, context)
	} else if operation == "cluster" {
		yamlFile, err := ioutil.ReadFile(configFile)

		makeDir("templates")

		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}

		SetupCluster.CheckCluster(yamlFile)
	} else if operation == "take_backup" {
		takeBackup(name, context)
		fmt.Print("Work In Progress\n")
	} else {
		fmt.Print("Operation Not Supported")
	}

	var manageOperation = StrSlice{"all", "init", "namespace", "storage", "resourcequota", "defaultquota", "serviceaccount"}

	if manageOperation.Has(operation) {
		filePath := "K8Cli" + "/mgmt/" + context

		ConfigFile := strings.TrimSpace(filePath + "/config.yaml")
		fileConfigYml, err := ioutil.ReadFile(ConfigFile)
		if err != nil {
			fmt.Println(err)
		}

		//var InitClusterConfigVals InitClusterConfigVals
		err = yaml.Unmarshal([]byte(fileConfigYml), &InitialConfigVals)
		if err != nil {
			panic(err)
		}


		if kubeconfigfile == "" {
			dirname, err := os.UserHomeDir()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(dirname)

			kubeconfigfile = dirname + "/.kube/config"
		}
		masterurl = getClusterEndpoint(context, kubeconfigfile)

	}

	if operation == "all" {

		connection := setupK8sConnection(InitialConfigVals)

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

		connection := setupK8sConnection(InitialConfigVals)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "storage" {

		connection := setupK8sConnection(InitialConfigVals)
		fmt.Println("Executing Create or Update StorageClasses")
		manageCluster.CreateorUpdateStorageClass(InitialConfigVals.ClusterDetails.StorageClassFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "resourcequota" {

		connection := setupK8sConnection(InitialConfigVals)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update DefaultQuotas")
		manageCluster.CreateorUpdateDefaultQuota(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update ResourceQuotas")
		manageCluster.CreateorUpdateResourceQuota(InitialConfigVals.ClusterDetails.ResourceQuotaFile, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "defaultquota" {

		connection := setupK8sConnection(InitialConfigVals)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update DefaultQuotas")
		manageCluster.CreateorUpdateDefaultQuota(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "serviceaccount" {

		connection := setupK8sConnection(InitialConfigVals)
		fmt.Println("Executing Create or Update NameSpaces")
		manageCluster.CreateorUpdateNameSpace(InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Println("Executing Create or Update NameSpaceUsers")
		manageCluster.CreateorUpdateNameSpaceUser(InitialConfigVals.ClusterDetails.Configs, InitialConfigVals.ClusterDetails.NameSpaceFile, connection, InitialConfigVals.ClusterDetails.MasterKey)

	} else if operation == "init" {

		fmt.Println("Initializing K8Cli")
		fmt.Printf("ClusterName: %v\n", &context)
		fmt.Printf("masterurl: %v\n", masterurl)
		fmt.Printf("kubeconfigfile: %v\n", kubeconfigfile)
		manageCluster.Init(context, masterurl, kubeconfigfile)

	} else {

		fmt.Printf("MasterUrl: %v\n", InitialConfigVals.ClusterDetails.MasterUrl)
		fmt.Printf("KubeConfig: %v\n", InitialConfigVals.ClusterDetails.KubeConfig)
		fmt.Printf("MasterKey: %v\n", InitialConfigVals.ClusterDetails.MasterKey)
		fmt.Printf("Configs: %v\n", InitialConfigVals.ClusterDetails.Configs)
		fmt.Printf("StorageClasses.yaml: %v\n", InitialConfigVals.ClusterDetails.StorageClassFile)
		fmt.Printf("Namepaces.yaml: %v\n", InitialConfigVals.ClusterDetails.NameSpaceFile)
		fmt.Printf("ResourceQuotas.yaml: %v\n", InitialConfigVals.ClusterDetails.ResourceQuotaFile)
		fmt.Println("Provide Valid input operation")
	}

	deleteDir("templates")
}
