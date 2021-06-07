package main

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
)

func Init(clustername string, kubeconfig string) (err error) {

	// Variables - host, namespace
	path := "K8Cli/" + clustername

	configpath := "K8Cli/" + clustername + "/mgmt" + "/configs"
	storageclasspath := "K8Cli/" + clustername + "/mgmt" + "/StorageClasses"
	namespacepath := "K8Cli/" + clustername + "/mgmt" + "/NameSpaces"
	resourcequotapath := "K8Cli/" + clustername + "/mgmt" + "/ResourceQuotas"

	storageclassfile := storageclasspath + "/StorageClasses.yml"
	namespacefile := namespacepath + "/Namespaces.yml"
	resourcequotafile := resourcequotapath + "/ResourceQuota.yml"

	addonsfilePath := "K8Cli/" + clustername + "/addons"
	clusterfilePath := "K8Cli/" + clustername + "/cluster"
	securitygppath := "K8Cli/" + clustername + "/SecurityGroups/"
	clusterpath := clusterfilePath + "/cluster.yml"
	sgpath := securitygppath + "/samplesg.yml"
	addonpath := addonsfilePath + "/addons.yml"

	//println(data)
	_, err = os.Stat(path)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(path, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		b := make([]byte, 4)
		rand.Read(b)
		token := fmt.Sprintf("%x", b)
		fmt.Println("Token generated: ", token)
		//type AutoGenerated struct {

		type ClusterDetails struct {
			ClusterName       string `yaml:"ClusterName"`
			MasterKey         string `yaml:"Masterkey"`
			KubeConfig        string `yaml:"Kubeconfig"`
			Configs           string `yaml:"Configs"`
			StorageClassFile  string `yaml:"StorageClassfile"`
			NameSpaceFile     string `yaml:"NameSpacefile"`
			ResourceQuotaFile string `yaml:"ResourceQuotafile"`
			ClusterYaml       string `yaml:"ClusterYaml"`
			Addons            string `yaml:"Addons"`
			SecurityGroups    string `Yaml:"SecurityGroups"`
		}
		var data = `
---
ClusterDetails:
  ClusterName: {{ .ClusterName }}
  MasterKey: {{ .MasterKey }}
  kubeConfig: {{ .KubeConfig }}
  Configs: {{ .Configs }}
  StorageClassFile: {{ .StorageClassFile }}
  NameSpaceFile: {{ .NameSpaceFile }}
  ResourceQuotaFile: {{ .ResourceQuotaFile }}
  ClusterYaml: {{ .ClusterYaml }}
  Addons: {{ .Addons }}
  SecurityGroups: {{ .SecurityGroups }}
`

		// Create the file:
		err = ioutil.WriteFile(path+"/config.tmpl", []byte(data), 0644)
		check(err)

		values := ClusterDetails{ClusterName: clustername, MasterKey: token, KubeConfig: kubeconfig, Configs: configpath, StorageClassFile: storageclassfile, NameSpaceFile: namespacefile, ResourceQuotaFile: resourcequotafile, Addons: addonpath, ClusterYaml: clusterpath}

		var templates *template.Template
		var allFiles []string

		if err != nil {
			fmt.Println(err)
		}

		//for _, file := range files {

		filename := "config.tmpl"
		fullPath := path + "/config.tmpl"
		if strings.HasSuffix(filename, ".tmpl") {
			allFiles = append(allFiles, fullPath)
		}
		//}
		fmt.Println(allFiles)
		templates, err = template.ParseFiles(allFiles...)
		if err != nil {
			fmt.Println(err)
		}

		s1 := templates.Lookup("config.tmpl")
		//f, err := os.Create(filePath+"/config.yml")
		f, err := os.Create(path + "/config.yml")
		if err != nil {
			panic(err)
		}

		//defer f.Close() // don't forget to close the file when finished.

		fmt.Println("Creating K8Cli folder and config files")
		// Write template to file:
		err = s1.Execute(f, values)
		defer f.Close() // don't forget to close the file when finished.
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("K8Cli/" + clustername + " exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(configpath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(configpath, 0755)

		var DefaultQuota = `
---
DefaultQuota:
  Details:
    - Name: Container
      max:
        cpu: 2
        memory: 1Gi
      min:
        cpu: 100m
        memory: 4Mi
      default:
        cpu: 300m
        memory: 200Mi
      defaultRequest:
        cpu: 200m
        memory: 100Mi
      maxLimitRequestRatio:
        cpu: 10
    - Name: Pod
      max:
        cpu: 2
        memory: 1Gi
      min:
        cpu: 200m
        memory: 6Mi
  Labels:
   Key1: Val1
   Key2: Val2
`
		var DefaultRole = `
---
NameSpaceRoleDetails:
  AppendName: test123
  PolicyRules:
    - APIGroups: ["","extensions","apps"]
      Resources: ["*"]
      Verbs:     ["*"]
    - APIGroups: ["batch"]
      Resources: ["jobs","cronjobs"]
      Verbs:     ["*"]
  Labels:
    Key1: Val1`

		fmt.Println("Creating K8Cli/" + clustername + "/mgmt/configs and sample files")
		err = ioutil.WriteFile(configpath+"/DefaultQuota.yml", []byte(DefaultQuota), 0644)
		check(err)
		err = ioutil.WriteFile(configpath+"/DefaultNameSpaceRole.yml", []byte(DefaultRole), 0644)
		check(err)

		if errDir != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Println("K8Cli/" + clustername + "/mgmt/configs exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(storageclasspath)
	if os.IsNotExist(err) {

		errDir := os.MkdirAll(storageclasspath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleStorageYaml = `
---
Delete:
  Enable: True
StorageClasses:
  - Name:
    Provisioner:
    Parameters:
      Key1 : Val1
    ReclaimPolicy:
    VolumeBindingMode:
    Labels:
      Key1: Val1
      Key2: Val2
  - Name: slow
    Provisioner: kubernetes.io/azure-disk
    Parameters:
      skuName: Standard_LRS
      location: eastus
      storageAccount: azure_storage_account_test
    ReclaimPolicy:
    VolumeBindingMode:
    Labels:
      Key1: Val1
      Key2: Val2
`
		fmt.Println("Creating K8Cli/" + clustername + "/mgmt/StorageClasses folder and sample files")
		err = ioutil.WriteFile(storageclassfile, []byte(SampleStorageYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/mgmt/StorageClasses exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(namespacepath)
	if os.IsNotExist(err) {

		errDir := os.MkdirAll(namespacepath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleNameSpaceYaml = `
---
Delete:
  Enable: True
NameSpace:
  - Name: "test"
    ResourceQuota: "q1"
    DefaultQuota: " "
    Labels:
      "Key1": "Val1"
  - Name: "test1"
    ResourceQuota: "q1"
    DefaultQuota: " "
    Labels:
      "Key1": "Val1"
`

		fmt.Println("Creating K8Cli/" + clustername + "/mgmt/NameSpaces folder and sample files")
		err = ioutil.WriteFile(namespacefile, []byte(SampleNameSpaceYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/mgmt/NameSpaces exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(resourcequotapath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(resourcequotapath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleResourceYaml = `
---
ResourceQuota:
  - QuotaName: "q1"
    RequestsCPU: 10
    LimitsCPU: 10
    RequestsMemory: 10
    LimitsMemory: 10
    Pods: 40
    RequestsStorage: 10
    RequestsEphemeralStorage: 10
    LimitsEphemeralStorage: 10
    StorageClasses:
      - Name:
        RequestsStorage: 5G
      - Name:
        RequestsStorage: 20G
      - Name:
        RequestsStorage: 40G
    Labels:
      "Key1": "Val1"
      "Key2": "Val2"
`
		fmt.Println("Creating K8Cli/" + clustername + "/mgmt/ResourceQuotas folder and sample files")
		err = ioutil.WriteFile(resourcequotafile, []byte(SampleResourceYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/mgmt/ResourceQuotas exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(securitygppath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(securitygppath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleSGYaml = `
Name: testsg1
Egress:
  - FromPort: 20
    IPProtocal: tcp
    IPRange: [193.0.2.0/24,198.51.100.0/24]
    ToPort: 35
  #SecurityGroups: [sg-123,sg-123]
  - FromPort: 31
    IPProtocal: udp
    IPRange: [0.0.0.0/0]
    ToPort: 35
  # - FromPort: -1
  #   IPProtocal: "-1"
  #   IPRange: [ 198.52.100.0/24 ]
  #   ToPort: -1
  #  #SecurityGroups: [ ]
Ingress:
  - FromPort: 20
    IPProtocal: tcp
    IPRange: [193.0.2.0/24,198.51.100.0/24]
    ToPort: 35
  #SecurityGroups: [sg-123,sg-123]
  - FromPort: 31
    IPProtocal: udp
    IPRange: [0.0.0.0/0]
    ToPort: 35
  # - FromPort: -1
  #   IPProtocal: "-1"
  #   IPRange: [ 198.52.100.0/24 ]
  #   ToPort: -1
  #  #SecurityGroups: [ ]`
		fmt.Println("Creating K8Cli/" + clustername + "/cluster/securitygroups/samplesg.yml sample file")
		err = ioutil.WriteFile(sgpath, []byte(SampleSGYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/cluster exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(clusterfilePath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(clusterfilePath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleClusterYaml = `
---
Cloud:
  Name: AWS
  Profile: kv3
  Region: us-east-1
  Cluster: test-eks5
  Bucket: k8-cloud-setup-cluster
VPC:
  VpcBlock: 10.1.0.0/16
  PublicSubnets:
    PublicSubnet01Block: 10.1.1.0/24
    PublicSubnet02Block: 10.1.2.0/24
  PrivateSubnets:
    PrivateSubnet01Block: 10.1.4.0/24
    PrivateSubnet02Block: 10.1.5.0/24
Master:
  KubernetesVersion: 1.18
#  SecurityGroupIds: sg-091340d1dd5486d40
  SubnetIds: [PublicSubnet02Block, PublicSubnet01Block, PrivateSubnet01Block, PrivateSubnet02Block ]
Nodes:
#  - NodegroupName: nodegroup-1
#    SubnetIds: []
#    InstanceTypes: m5.large
#  - NodegroupName: nodegroup-2
#    SubnetIds: [PrivateSubnet01Block, PrivateSubnet02Block]
#    InstanceTypes: t2.small
  - NodegroupName: nodegroup-3
    SubnetIds: [ PrivateSubnet01Block, PrivateSubnet02Block ]
    InstanceTypes: t2.small`

		fmt.Println("Creating K8Cli/" + clustername + "/cluster/cluster.yml sample file")
		err = ioutil.WriteFile(clusterpath, []byte(SampleClusterYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/cluster exists, please manually edit file to make changes or provide new cluster name")
	}

	_, err = os.Stat(addonsfilePath)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(addonsfilePath, 0755)
		if errDir != nil {
			log.Fatal(err)
		}

		var SampleAddonYaml = `
#helm:
#  service_account: tiller-dev
#  tiller_namespace: dev #kube-system
#  major_version: 2
repositories:
  - name: stable
    url: https://charts.helm.sh/stable
  #  - name: jfrog
#    url:  https://charts.jfrog.io/
  - name: bitnami
    url:  https://charts.bitnami.com/bitnami
releases:
  - name: kube-state-metrics
    namespace: kube-system
    version: 2.8.4
    chart: stable/kube-state-metrics
#    values_file: /tmp/prometheus.yml
  - name: wordpress
    version: 9.3.10
    chart: bitnami/wordpress
    values_file: examples/cluster-1/wordpress.yml
  - name: velero
    chart: "vmware-tanzu/velero"
    version: "2.12.0"
    values_file: examples/cluster-1/velero.yml`

		fmt.Println("Creating K8Cli/" + clustername + "/addons/addons.yml sample file")
		err = ioutil.WriteFile(addonpath, []byte(SampleAddonYaml), 0644)
		check(err)

	} else {
		fmt.Println("K8Cli/" + clustername + "/addons exists, please manually edit file to make changes or provide new cluster name")
	}

	return
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}