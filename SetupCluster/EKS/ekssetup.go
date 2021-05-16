package ekssetup

import (
	_ "bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"net/http"
	_ "net/http"
	//"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awscf "github.com/aws/aws-sdk-go/service/cloudformation"
	_ "github.com/aws/aws-sdk-go/service/s3"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Key   string
	Value string
}
type cftvpc struct {
	StackName   string
	TemplateURL string
}
type YamlConvert struct {
	data interface{}
}
type eksSession struct {
	Cloud struct {
		Profile      string `yaml:"Profile"`
		AccessKey    string `yaml:"AccessKey"`
		SecretAccKey string `yaml:"SecretAccKey"`
		Region       string `yaml:"Region"`
		Cluster      string `yaml:"Cluster"`
		Bucket       string `yaml:"Bucket"`
	} `yaml:"Cloud"`
}
type EksVPC struct {
	VPC struct {
		VpcBlock       string                      `yaml:"VpcBlock"`
		PublicSubnets  map[interface{}]interface{} `yaml:"PublicSubnets"`
		PrivateSubnets map[interface{}]interface{} `yaml:"PrivateSubnets"`
	} `yaml:"VPC"`
}
type EksMaster struct {
	Master struct {
		KubernetesVersion string      `yaml:"KubernetesVersion"`
		SecurityGroupIds  string      `yaml:"SecurityGroupIds"`
		SubnetIds         interface{} `yaml:"SubnetIds"`
	} `yaml:"Master"`
}
type NodeList struct {
	Nodes []Nodevalues `yaml:"Nodes"`
}
type Nodevalues struct {
	NodegroupName string      `yaml:"NodegroupName"`
	InstanceTypes string      `yaml:"InstanceTypes"`
	SubnetIds     interface{} `yaml:"SubnetIds"`
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func getFileFromURL(fileName string, fileUrl string)  {
	err := DownloadFile(fileName, fileUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println("Downloaded: " + fileUrl)

}

//Setup EKS Cluster

func ReadEKSYaml(f []byte) {
	////Setting up variables
	ElementsSubnetIDs := make(map[string]string)

	var MClusterName, vpcsubnets, vpcsecuritygps, vpcclustername, MSubnetIds, Profile, Acceesskey, Secretkey, Region, Cluster, VPCfileName, EksfileName, NodesfileName,  VPCSourceFile string
	var nodelen int

	var sess *session.Session
	var eksSession eksSession
	var eksvpc EksVPC
	var eksMaster EksMaster
	var ConfNode NodeList

	////Reading inputs from yaml
	file := f

	err := yaml.Unmarshal([]byte(file), &eksSession)
	if err != nil {
		panic(err)
	}

	//Start EKS Cluster elements
	//fmt.Printf("AccessKey: %v\n", eksSession.Cloud.AccessKey)
	//fmt.Printf("SecretKey: %v\n", eksSession.Cloud.SecretAccKey)
	fmt.Println("Setting up EKS Cluster ........")

	Profile = eksSession.Cloud.Profile
	Acceesskey = eksSession.Cloud.AccessKey
	Secretkey = eksSession.Cloud.SecretAccKey
	Region = eksSession.Cloud.Region
	Cluster = eksSession.Cloud.Cluster

	fmt.Printf("Creating sessions.......")

	//Create AWS session
	if Profile == "" {
		sess, err = session.NewSession(&aws.Config{
			//aws.Config{throttle.Throttle()}
			Region:      aws.String(Region),
			Credentials: credentials.NewStaticCredentials(Acceesskey, Secretkey, ""),
		})
	} else {
		sess, err = session.NewSessionWithOptions(session.Options{
			// Specify profile to load for the session's config
			Profile: Profile,

			// Provide SDK Config options, such as Region.
			Config: aws.Config{
				Region: aws.String(Region),
			},

			// Force enable Shared Config support
			//SharedConfigState: session.SharedConfigEnable,
		})
	}
	fmt.Printf("Session created \n")

	////Setting up S3 Bucket
	fmt.Printf("Setting up S3 bucket \n")
	S3Name := eksSession.Cloud.Bucket

	//Loading Yaml
	VPCFile, err := ioutil.ReadFile("templates/0001-vpc.yaml")
	EKSFile, err := ioutil.ReadFile("templates/0005-eks-cluster.yaml")
	NodeFile, err := ioutil.ReadFile("templates/0007-esk-managed-node-group.yaml")
	//Add Yaml templates to s3
	err, VPCfileName, EksfileName, NodesfileName = AddFileToS3(sess, VPCFile, EKSFile, NodeFile, S3Name, Cluster)
	if err != nil {
		log.Fatal(err)
	}

	////Checking if VPC is enabled
	fmt.Printf("Checking if VPC creation enabled\n")
	err = yaml.Unmarshal([]byte(file), &eksvpc)
	if err != nil {
		panic(err)
	}
	VPCName := eksvpc.VPC.VpcBlock
	PublicSubnetLen := len(eksvpc.VPC.PublicSubnets)
	PrivateSubnetLen := len(eksvpc.VPC.PrivateSubnets)

	if PublicSubnetLen != PrivateSubnetLen {
		fmt.Printf("PublicSubnets and PrivateSubnets count should  be same\n")
		os.Exit(255)
	}

	if PublicSubnetLen == 2 {
		VPCSourceFile = "https://k8s-cloud-templates.s3.amazonaws.com/vpc-4subnets.yaml"
	} else if PublicSubnetLen == 3 {
		VPCSourceFile = "https://k8s-cloud-templates.s3.amazonaws.com/vpc-6subnets.yaml"
	}

	getFileFromURL("templates/0001-vpc.yaml",VPCSourceFile)
	getFileFromURL("templates/0005-eks-cluster.yaml","https://k8s-cloud-templates.s3.amazonaws.com/0005-eks-cluster.yaml")
	getFileFromURL("templates/0007-esk-managed-node-group.yaml","https://k8s-cloud-templates.s3.amazonaws.com/0007-esk-managed-node-group.yaml")


	if VPCName != "" {
		fmt.Printf("VPC creation enabled, creating/updating VPC.......\n")
		vpcsubnets, vpcsecuritygps, vpcclustername, ElementsSubnetIDs = Create_VPC(sess, file, Cluster, S3Name, VPCfileName)
	} else {
		fmt.Printf("VPC creation Not Enabled\n")
	}

	err = yaml.Unmarshal([]byte(file), &ConfNode)
	var count = len(ConfNode.Nodes)
	nodelen = count
	//	println("Value of node length in first go", nodelen)

	//Checking if Master Cluster creation is enabled
	err = yaml.Unmarshal([]byte(file), &eksMaster)
	if err != nil {
		panic(err)
	}
	MasterName := eksMaster.Master.KubernetesVersion
	if MasterName != "" {
		fmt.Printf("Master creation enabled, creating/updating stacks.......\n")
		MClusterName, MSubnetIds = Create_Master(sess, vpcsecuritygps, vpcclustername, vpcsubnets, ElementsSubnetIDs, file, Cluster, S3Name, EksfileName)
		if nodelen == 0 {
			fmt.Printf("Master creation completed, no node groups provided.......\n")
		} else if nodelen != 0 {
			fmt.Printf("Master creation completed, node groups listed.......\n")
			fmt.Printf("Creating node groups.......\n")
			for i := 0; i < nodelen; i++ {
				println("Subnets passed from Master: ", MSubnetIds)
				println("Cluster Name passed from Master: ", MClusterName)
				Create_Node(sess, i, MClusterName, MSubnetIds, ElementsSubnetIDs, file, Cluster, S3Name, NodesfileName)
			}
		}
	} else {
		fmt.Printf("EKS Cluster Not Enabled\n")
	}

}
func AddFileToS3(sess *session.Session, VPC []byte, EKS []byte, Nodes []byte, s3 string, cluster string) (error, string, string, string) {
	svc := s3manager.NewUploader(sess)

	// Open the file for use
	VPCfile := string(VPC)
	VPCfileName := cluster + "-VPC" + ".yml"
	println("VPC Cloudformation YAML Name: \n", VPCfileName)
	Eksfile := string(EKS)
	EksfileName := cluster + "-EKS" + ".yml"
	println("EKS Cluster Cloudformation YAML Name: \n", EksfileName)
	Nodefile := string(Nodes)
	NodesfileName := cluster + "-Nodes" + ".yml"
	println("Nodes Cloudformation YAML Name: \n", NodesfileName)

	_, err := svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3),             // Bucket to be used
		Key:    aws.String(VPCfileName),    // Name of the file to be saved
		Body:   strings.NewReader(VPCfile), // File
	})
	if err != nil {
		fmt.Println(err)
		return err, "", "", ""
	}
	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3),             // Bucket to be used
		Key:    aws.String(EksfileName),    // Name of the file to be saved
		Body:   strings.NewReader(Eksfile), // File
	})
	if err != nil {
		// Do your error handling here
		return err, "", "", " "
	}
	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3),              // Bucket to be used
		Key:    aws.String(NodesfileName),   // Name of the file to be saved
		Body:   strings.NewReader(Nodefile), // File
	})
	if err != nil {
		// Do your error handling here
		return err, "", "", " "
	}

	return err, VPCfileName, EksfileName, NodesfileName
}
func Create_VPC(sess *session.Session, file []byte, cluster string, S3 string, VPCFilename string) (string, string, string, map[string]string) {

	// Creating vars
	fileVPC := file
	var eksvpc EksVPC

	ElementsSubnetIDs := make(map[string]string)
	ElementsCreate := make(map[string]string)
	ElementsUpdate := make(map[string]string)
	ElementsSubnets := make(map[string]string)

	var v = cftvpc{}
	var value, Keyname string
	var vpcsubnets string
	var vpcsecuritygps string
	var vpcclustername string
	err := yaml.Unmarshal([]byte(fileVPC), &eksvpc)

	//StackName := eksvpc.VPC.StackName
	StackName := cluster + "-VPC-Stack"
	VpcBlock := eksvpc.VPC.VpcBlock
	ClusterName := cluster
	//ClusterName := eksvpc.VPC.ClusterName

	ElementsCreate = map[string]string{
		"VpcBlock":    VpcBlock,
		"ClusterName": ClusterName,
	}
	ElementsUpdate = map[string]string{}

	datapublic := eksvpc.VPC.PublicSubnets
	//	fmt.Printf("Checking DataPublic: %v\n", datapublic)
	keys := make([]string, 0)
	for KEY, _ := range datapublic {
		if s, ok := KEY.(string); ok {
			keys = append(keys, s)
		}
	}
	//PublicSubnetKeys := yaml.Get("VPC").Get("PublicSubnets").GetMapKeys()
	//PublicSubnet, _ := yaml.Get("VPC").Get("PublicSubnets").Map()
	PublicSubnetKeys := keys
	//fmt.Printf(PublicSubnetKeys)
	PublicSubnet := datapublic
	NoofKeyspublic := len(PublicSubnetKeys)
	//	fmt.Printf("No of public Key: %v\n", NoofKeyspublic)
	for i := 0; i < NoofKeyspublic; i++ {
		Keyname = PublicSubnetKeys[i]
		//		fmt.Printf("KeyName: %v\n", Keyname)
		fmt.Printf("Keyname passed: %v\n", PublicSubnetKeys[i])
		value, _ = strconv.Unquote(awsutil.StringValue(PublicSubnet[Keyname]))
		fmt.Printf("Values prased: %#v\n", value)
		ElementsCreate[Keyname] = value
		ElementsSubnets[Keyname] = value
	}

	dataprivate := eksvpc.VPC.PrivateSubnets
	keys = make([]string, 0)
	for k, _ := range dataprivate {
		if s, ok := k.(string); ok {
			keys = append(keys, s)
		}
	}
	PrivateSubnetKeys := keys
	PrivateSubnet := dataprivate

	//PrivateSubnetKeys, _ := yaml.Get("VPC").Get("PrivateSubnets").GetMapKeys()
	//fmt.Printf(PrivateSubnetKeys)
	//PrivateSubnet, _ := yaml.Get("VPC").Get("PrivateSubnets").Map()

	NoofKeysprivate := len(PrivateSubnetKeys)
	for i := 0; i < NoofKeysprivate; i++ {
		Keyname = PrivateSubnetKeys[i]
		//fmt.Printf(Keyname)
		//fmt.Printf(PrivateSubnetKeys[i])
		value, _ = strconv.Unquote(awsutil.StringValue(PrivateSubnet[Keyname]))
		//fmt.Printf(value)
		ElementsCreate[Keyname] = value
		ElementsSubnets[Keyname] = value
		//ElementsUpdate[Keyname] = value not updating VPC after it is created
	}

	//TemplateURL, _ := yaml.Get("VPC").Get("TemplateURL").String()
	//TemplateURL := eksvpc.VPC.TemplateURL
	TemplateURL := "https://" + S3 + ".s3.amazonaws.com/" + VPCFilename
	v.StackName = StackName
	v.TemplateURL = TemplateURL

	//Passing values for creating stack

	//	fmt.Println(".......ElementsCreate.....", ElementsCreate)
	//Passing values for updating Stack

	//	fmt.Println(".......ElementsUpdate.....", ElementsUpdate)
	fmt.Printf("StackName: %v\n", v.StackName)
	fmt.Printf("TemplateURL: %v\n", v.TemplateURL)

	if err != nil {
		fmt.Println(os.Stderr, "YAML Prasing failed with Error: %v\n", err)
		os.Exit(1)
	}

	// Calling stack validation

	a, b := ValidateStack(sess, v.TemplateURL, ElementsCreate, ElementsUpdate)

	// Calling outputs from created/updated stack

	ListStack(sess, v, a, b)

	NoOP := len(CheckStack(sess, StackName).Stacks[0].Outputs)

	for p := 0; p < NoOP; p++ {
		//time.Sleep(5 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "SubnetIds" {
			//time.Sleep(5 * time.Second)
			value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
			fmt.Printf("Subnets: %v\n", value)
			vpcsubnets = value
			//time.Sleep(5 * time.Second)
		}
	}
	for p := 0; p < NoOP; p++ {
		//time.Sleep(5 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "SecurityGroups" {
			//time.Sleep(5 * time.Second)
			value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
			fmt.Printf("SecurityGroups: %v\n", value)
			vpcsecuritygps = value
			//time.Sleep(5 * time.Second)
		}
	}
	for p := 0; p < NoOP; p++ {
		//time.Sleep(5 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "ClusterName" {
			//time.Sleep(5 * time.Second)
			value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
			fmt.Printf("Cluster Name: %v\n", value)
			vpcclustername = value
			//time.Sleep(5 * time.Second)
		}
	}

	// Creating SubnetIDs elements

	for i := 0; i < NoofKeysprivate; i++ {
		Keyname = PrivateSubnetKeys[i]
		for p := 0; p < NoOP; p++ {
			//time.Sleep(5 * time.Second)
			k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
			var c string = strings.Trim(k, "\"")
			if string(c) == Keyname {
				//time.Sleep(5 * time.Second)
				value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
				//fmt.Printf(Keyname, ":", value)
				fmt.Printf("%v", Keyname)
				fmt.Printf(":")
				fmt.Printf("%v\n", value)
				ElementsSubnetIDs[strconv.Quote(Keyname)] = value
				//time.Sleep(5 * time.Second)
			}
		}
	}
	for i := 0; i < NoofKeyspublic; i++ {
		Keyname = PublicSubnetKeys[i]
		for p := 0; p < NoOP; p++ {
			//time.Sleep(5 * time.Second)
			k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
			var c string = strings.Trim(k, "\"")
			if string(c) == Keyname {
				//time.Sleep(5 * time.Second)
				value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
				fmt.Printf("%v", Keyname)
				fmt.Printf(":")
				fmt.Printf("%v\n", value)
				ElementsSubnetIDs[strconv.Quote(Keyname)] = value
				//time.Sleep(5 * time.Second)
			}
		}
	}

	//	fmt.Printf("ElementsSubnetIDs: %v\n", ElementsSubnetIDs)
	list := CheckStack(sess, StackName).Stacks[0].StackName
	fmt.Printf("StackID of the Stack: %v\n", awsutil.StringValue(list))
	if err != nil {
		panic(err)
	}

	return vpcsubnets, vpcsecuritygps, vpcclustername, ElementsSubnetIDs
}
func Create_Master(sess *session.Session, vpcsecuritygps string, vpcclustername string, vpcsubnets string, ElementsSubnetIDs map[string]string, file []byte, cluster string, s3 string, eksfileName string) (string, string) {

	// Creating vars
	//svc := awscf.New(sess)
	ElementsCreate := make(map[string]string)
	ElementsUpdate := make(map[string]string)
	var v = cftvpc{}
	var ClusterName, SecurityGroupIds, SubnetIds string
	var eksmaster EksMaster
	FileMaster := file
	err := yaml.Unmarshal([]byte(FileMaster), &eksmaster)

	//StackName := eksmaster.Master.StackName
	//TemplateURL := eksmaster.Master.TemplateURL
	TemplateURL := "https://" + s3 + ".s3.amazonaws.com/" + eksfileName
	StackName := cluster + "-EKS-Stack"
	KubernetesVersion := eksmaster.Master.KubernetesVersion
	//NodesSelected := eksmaster.Master.Nodes

	if vpcclustername == "" {
		//ClusterName = eksmaster.Master.ClusterName
		ClusterName = cluster
	} else if vpcclustername != "" {
		ClusterName = strings.Trim(vpcclustername, "\"")
	}
	if vpcsecuritygps == "" {
		SecurityGroupIds = eksmaster.Master.SecurityGroupIds
	} else if vpcsecuritygps != "" {
		SecurityGroupIds = strings.Trim(vpcsecuritygps, "\"")
	}
	if vpcsubnets == "" {
		SubnetIds = eksmaster.Master.SubnetIds.(string)
	} else if vpcsubnets != "" {
		arrayl := eksmaster.Master.SubnetIds.([]interface{})
		//		fmt.Printf("Check Arryl ...\n", arrayl)
		arrlen := len(arrayl)
		//		fmt.Printf("Array Lenght...\n", arrlen)
		arropt := make([]string, int(arrlen))
		if arrlen == 0 {
			SubnetIds = strings.Trim(vpcsubnets, "\"")
		} else if arrlen != 0 {
			for i := 0; i < arrlen; i++ {
				var subnetIDValue string
				subnetName := awsutil.StringValue(arrayl[i])
				b := strconv.Quote(strings.Trim(subnetName, "\""))
				if ElementsSubnetIDs[b] != "" {
					subnetIDValue = ElementsSubnetIDs[b]
				} else if ElementsSubnetIDs[b] == "" {
					subnetIDValue = string(b)
				}
				arropt[i] = subnetIDValue
			}
			//ArraySubnetIds = arropt
			SubnetIds, _ = strconv.Unquote(awsutil.StringValue(strings.Join(arropt, ",")))
		}
	}

	fmt.Printf("Values getting passed: %v\n", SubnetIds, SecurityGroupIds, ClusterName)

	SubnetIdsTrim := strings.TrimSpace(strings.Trim(strings.Trim(strings.Trim(SubnetIds, "\""), "\""), ""))
	SubnetIdsReplace := strings.Replace(SubnetIdsTrim, "\",\"", ",", -1)
	v.StackName = StackName
	v.TemplateURL = TemplateURL

	//Passing values for creating stack
	ElementsCreate = map[string]string{
		"ClusterName":       ClusterName,
		"KubernetesVersion": KubernetesVersion,
		"SecurityGroupIds":  SecurityGroupIds,
		"SubnetIds":         SubnetIdsReplace,
	}
	//	fmt.Printf("Create Elements: %v\n", ElementsCreate)

	//Passing values for updating Stack

	ElementsUpdate = map[string]string{
		"ClusterName":       ClusterName,
		"KubernetesVersion": KubernetesVersion,
		"SecurityGroupIds":  SecurityGroupIds,
		"SubnetIds":         SubnetIdsReplace,
	}
	//	fmt.Printf("Update Elements: %v\n", ElementsUpdate)
	fmt.Printf("StackName: %v\n", awsutil.StringValue(v.StackName))
	fmt.Printf("TemplateURL: %v\n", v.TemplateURL)

	if err != nil {
		fmt.Println(os.Stderr, "YAML Prasing failed with Error: %v\n", err)
		os.Exit(1)
	}

	// Calling stack validation

	a, b := ValidateStack(sess, v.TemplateURL, ElementsCreate, ElementsUpdate)

	// Calling outputs from created/updated stack

	ListStack(sess, v, a, b)
	list := CheckStack(sess, StackName).Stacks[0].StackName

	//NoOP := len(CheckStack(sess, StackName).Stacks[0].Outputs)
	fmt.Printf("StackID of the Stack: %v\n", awsutil.StringValue(list))
	if err != nil {
		panic(err)
	}

	//Nodelen := len(NodesSelected)
	//return Nodelen, NodesSelected, ClusterName, SubnetIdsReplace
	return ClusterName, SubnetIdsReplace

}
func Create_Node(sess *session.Session, nodelen int, MClusterName string, MSubnetIds string, ElementsSubnetIDs map[string]string, f []byte, cluster string, s3 string, nodesfileName string) {

	// Creating vars
	file := f
	var v = cftvpc{}
	//var NodeName,
	var NodeClusterName, NSubnetIds, NodeTemplateURL, NodegroupName, NodeStackName, InstanceTypes string

	var ConfNode NodeList
	err := yaml.Unmarshal([]byte(file), &ConfNode)
	if err != nil {
		panic(err)
	}

	ElementsCreate := make(map[string]string)
	ElementsUpdate := make(map[string]string)

	//fmt.Println("Creating Node group: ", NodeName)
	fmt.Println("Master Cluster SubnetIds: ", MSubnetIds)

	err = yaml.Unmarshal([]byte(file), &ConfNode)
	//err = yaml.Unmarshal([]byte(file), &eksnodes)
	if err != nil {
		panic(err)
	}

	TemplateURL := "https://" + s3 + ".s3.amazonaws.com/" + nodesfileName
	StackName := cluster + "-Node-Stack-" + strconv.Itoa(nodelen)

	//NodeStackName = ConfNode.Nodes[int(nodelen)].StackName
	//NodeTemplateURL = ConfNode.Nodes[int(nodelen)].TemplateURL
	NodeStackName = StackName
	NodeTemplateURL = TemplateURL
	NodegroupName = ConfNode.Nodes[int(nodelen)].NodegroupName
	InstanceTypes = ConfNode.Nodes[int(nodelen)].InstanceTypes
	//NodeSubnets = ConfNode.Nodes[int(nodelen)].SubnetIds

	fmt.Println("Node: 				 ", nodelen)
	//fmt.Println("Node Cluster Name:  ", NodeClusterName)
	fmt.Println("Node Stack Name: 	 ", NodeStackName)
	fmt.Println("Node Template Name: ", NodeTemplateURL)
	fmt.Println("Node Subnets: 	     ", MSubnetIds)
	fmt.Println("Node Group Name: 	 ", NodegroupName)

	if MClusterName == "" {
		//NodeClusterName = ConfNode.Nodes[int(nodelen)].ClusterName
		NodeClusterName = cluster
	} else if MClusterName != "" {
		NodeClusterName = MClusterName
	}

	if InstanceTypes == "" {
		//NodeClusterName = ConfNode.Nodes[int(nodelen)].ClusterName
		InstanceTypes = "t3.large"
	}

	if MSubnetIds == "" {
		NSubnetIds = ConfNode.Nodes[int(nodelen)].SubnetIds.(string)
	} else if MSubnetIds != "" {
		arrayl := ConfNode.Nodes[int(nodelen)].SubnetIds.([]interface{})
		fmt.Println("Subnets passed for the node: ", arrayl)
		arrlen := len(arrayl)
		arropt := make([]string, int(arrlen))
		if arrlen == 0 {
			NSubnetIds = strings.Trim(awsutil.StringValue(MSubnetIds), "\"")
		} else if arrlen != 0 {
			for i := 0; i < arrlen; i++ {
				var subnetIDValue string
				subnetName := awsutil.StringValue(arrayl[i])
				b := strconv.Quote(strings.Trim(subnetName, "\""))
				if ElementsSubnetIDs[b] != "" {
					subnetIDValue = ElementsSubnetIDs[b]
				} else if ElementsSubnetIDs[b] == "" {
					subnetIDValue = string(b)
				}
				arropt[i] = subnetIDValue
			}
			NSubnetIds, _ = strconv.Unquote(awsutil.StringValue(strings.Join(arropt, ",")))
		}
	}

	fmt.Println("Values getting passed: ", NSubnetIds, NodeClusterName, NodegroupName, InstanceTypes)
	//fmt.Printf(yaml.Get("VPC").Map())
	NodeSubnetIdsTrim := strings.TrimSpace(strings.Trim(strings.Trim(strings.Trim(NSubnetIds, "\""), "\""), ""))
	NodeSubnetIdsReplace := strings.Replace(NodeSubnetIdsTrim, "\",\"", ",", -1)
	v.StackName = NodeStackName
	v.TemplateURL = NodeTemplateURL

	//Passing values for creating stack
	ElementsCreate = map[string]string{
		"ClusterName":   NodeClusterName,
		"NodegroupName": NodegroupName,
		"SubnetIds":     NodeSubnetIdsReplace,
		"InstanceTypes": InstanceTypes,
		//"SecurityGroupIds": "sg-09972c390e1452989",
	}
	//	fmt.Println("Create Elements :", ElementsCreate)

	//Passing values for updating Stack

	ElementsUpdate = map[string]string{
		"ClusterName":   NodeClusterName,
		"NodegroupName": NodegroupName,
		"SubnetIds":     NodeSubnetIdsReplace,
		"InstanceTypes": InstanceTypes,
		//"SecurityGroupIds": "sg-09972c390e1452989",
	}
	//	fmt.Printf("Update Elements :", ElementsUpdate)
	fmt.Printf("StackName: %#v\n", v.StackName)
	fmt.Printf("TemplateURL: %#v\n", v.TemplateURL)

	if err != nil {
		fmt.Println(os.Stderr, "YAML Prasing failed with Error: %v\n", err)
		os.Exit(1)
	}

	// Calling stack validation

	a, b := ValidateStack(sess, v.TemplateURL, ElementsCreate, ElementsUpdate)

	// Calling outputs from created/updated stack

	ListStack(sess, v, a, b)
	list := CheckStack(sess, NodeStackName).Stacks[0].StackName

	//NoOP := len(CheckStack(sess, StackName).Stacks[0].Outputs)
	fmt.Println("StackID of the Stack:", awsutil.StringValue(list))
	if err != nil {
		panic(err)
	}

}
func ValidateStack(sess *session.Session, TemplateURL string, ElementsCreate map[string]string, ElementsUpdate map[string]string) ([]*awscf.Parameter, []*awscf.Parameter) {
	svc := awscf.New(sess)
	println("Validation session: ", svc)
	params := &awscf.ValidateTemplateInput{
		//TemplateBody: aws.String("TemplateBody"),
		TemplateURL: aws.String(TemplateURL),
	}
	resp, err := svc.ValidateTemplate(params)

	if err != nil {
		fmt.Println(os.Stderr, "Validation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Println("Stack validation passed")
	}
	fmt.Println("Stack passed: ", awsutil.StringValue(resp))
	//	fmt.Println("Number of Parameters defined in Stack: ", len(resp.Parameters))

	paramcreate := make([]*awscf.Parameter, len(resp.Parameters))
	paramupdate := make([]*awscf.Parameter, len(resp.Parameters))

	for i, p := range resp.Parameters {
		paramcreate[i] = &awscf.Parameter{
			ParameterKey: p.ParameterKey,
			////UsePreviousValue: aws.Bool(true),
			//Description: p.Description,
			//NoEcho:      p.NoEcho,
		}
		e := awsutil.StringValue(paramcreate[i].ParameterKey)
		k := strings.Trim(e, "\"")
		//println("test: ", elements[k])
		if ElementsCreate[k] != "" {
			paramcreate[i].ParameterValue = aws.String(ElementsCreate[k])
		} else {
			paramcreate[i].ParameterValue = p.DefaultValue
		}
	}

	for i, p := range resp.Parameters {
		paramupdate[i] = &awscf.Parameter{
			ParameterKey: p.ParameterKey,
			////UsePreviousValue: aws.Bool(true),
			//Description:  p.Description,
			//NoEcho:       p.NoEcho,
		}
		f := awsutil.StringValue(paramupdate[i].ParameterKey)
		l := strings.Trim(f, "\"")
		if ElementsUpdate[l] != "" {
			paramupdate[i].ParameterValue = aws.String(ElementsUpdate[l])
			paramupdate[i].UsePreviousValue = aws.Bool(false)
		} else {
			paramupdate[i].UsePreviousValue = aws.Bool(true)
		}
	}

	return paramcreate, paramupdate
}
func ListStack(sess *session.Session, c cftvpc, stackcreate []*awscf.Parameter, stackupdate []*awscf.Parameter) {
	type ByAge []Config
	var v = c
	var count = 0
	svc := awscf.New(sess)
	params := &awscf.DescribeStacksInput{}
	resp, err := svc.DescribeStacks(params)

	if err != nil {
		fmt.Println(os.Stderr, "Validation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Println("Checking Stacks.......")
	}

	value := awsutil.StringValue(len(resp.Stacks))
	i, _ := strconv.Atoi(value)

	//	fmt.Println("Number of Cloud Formation Templates exists:", i)

	if i == 0 {
		fmt.Println("No stacks exist, creating stack")
		Createcft(sess, v, stackcreate)
		//j := 0
		println("Checking Status.......")
		for {
			var a string = "CREATE_IN_PROGRESS"
			b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
			//print(b)
			var c string = strings.Trim(b, "\"")
			//var b string = strings.Trim(DescribeStack(sess, c.StackName, j), "\"")
			if a != c {
				fmt.Println("Status: ", b)
				//time.Sleep(5 * time.Second)
				fmt.Println()
				break
			}
		}
		fmt.Println("Creation Completed")

	} else if i != 0 {
		present := make([]Config, int(i))
		for k := 0; k < i; k++ {
			stacks, _ := strconv.Unquote(awsutil.StringValue(resp.Stacks[k].StackName))
			if stacks == c.StackName {
				present[k].Key = "yes"
			} else if stacks != c.StackName {
				present[k].Key = "no"
			}
		}
		fmt.Println(present)
		for i := range present {
			if present[i].Key == "yes" {
				count = 1
			} else if present[i].Key != "yes" {
			}
		}
		//		fmt.Println("Count :", count)
		if count == 1 {
			for k := 0; k < i; k++ {
				stacks, _ := strconv.Unquote(awsutil.StringValue(resp.Stacks[k].StackName))
				if stacks == c.StackName {
					//j := k
					fmt.Println("Stack exist, updating stack")
					UpdateStack(sess, v, stackupdate)
					println("Checking Status.......")
					for {
						var a string = "UPDATE_IN_PROGRESS"
						b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
						var c string = strings.Trim(b, "\"")
						if a != c {
							fmt.Println("Status: ", b)
							//time.Sleep(5 * time.Second)
							fmt.Println()
							break
						}
					}
					fmt.Println("Update Completed")
				}
			}
		} else if count == 0 {
			//j := 0
			fmt.Println("Stack doesn't exist, creating stack")
			Createcft(sess, v, stackcreate)
			println("Checking Status.......")
			for {
				var a string = "CREATE_IN_PROGRESS"
				b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
				//print(b)
				var c string = strings.Trim(b, "\"")
				//var b string = strings.Trim(DescribeStack(sess, c.StackName, j), "\"")
				if a != c {
					fmt.Println("Status: ", b)
					//time.Sleep(5 * time.Second)
					fmt.Println()
					break
				}
			}
			fmt.Println("Create Completed")
		}
	}
}
func CheckStack(sess *session.Session, StackName string) *awscf.DescribeStacksOutput {

	svc := awscf.New(sess)
	params := &awscf.DescribeStacksInput{
		StackName: aws.String(StackName),
	}
	resp, err := svc.DescribeStacks(params)

	if err != nil {
		fmt.Println(os.Stderr, "Listing failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		//fmt.Println("Listing stacks passed")
	}
	return resp
}
func Createcft(sess *session.Session, d cftvpc, stack []*awscf.Parameter) *awscf.CreateStackOutput {
	svc := awscf.New(sess)
	var params = &awscf.CreateStackInput{
		Capabilities: []*string{
			aws.String("CAPABILITY_IAM"),
		},
		ClientRequestToken:          nil,
		DisableRollback:             aws.Bool(false),
		EnableTerminationProtection: nil,
		NotificationARNs:            nil,
		OnFailure:                   nil,
		Parameters:                  stack,
		ResourceTypes:               nil,
		RoleARN:                     nil,
		RollbackConfiguration:       nil,
		StackName:                   aws.String(d.StackName),
		StackPolicyBody:             nil,
		StackPolicyURL:              nil,
		Tags:                        nil,
		TimeoutInMinutes:            nil,
		TemplateURL:                 aws.String(d.TemplateURL),
	}
	fmt.Println("Stack paramenters for creating stack :", params)
	rep, err := svc.CreateStack(params)

	if err != nil {
		fmt.Println(os.Stderr, "Creation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Println("Stack Creation passed")
	}

	//fmt.Printf(awsutil.StringValue(rep))

	return rep
}
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	//os.Exit(1)
}
func UpdateStack(sess *session.Session, u cftvpc, stack []*awscf.Parameter) *awscf.UpdateStackOutput {
	svc := awscf.New(sess)
	//fmt.Println(stack)
	params := &awscf.UpdateStackInput{
		StackName: aws.String(u.StackName),
		Capabilities: []*string{
			aws.String("CAPABILITY_IAM"),
		},
		NotificationARNs:            nil,
		Parameters:                  stack,
		StackPolicyBody:             nil,
		StackPolicyDuringUpdateBody: nil,
		StackPolicyDuringUpdateURL:  nil,
		StackPolicyURL:              nil,
		UsePreviousTemplate:         aws.Bool(true),
	}
	fmt.Println("Stack paramenters for updating stack :", params)
	resp, err := svc.UpdateStack(params)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case awscf.HandlerErrorCodeThrottling:
				exitErrorf("Throttling Error", os.Args[1])
			case awscf.ErrCodeChangeSetNotFoundException:
				exitErrorf("No updates to be performed", os.Args[2], os.Args[1])
			}
		}
		exitErrorf("unknown error occurred, %v", err)
	}

	//fmt.Printf("StackID: ", awsutil.StringValue(resp.StackId))
	return resp

}
func readJSON(path string) (*map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	contents := make(map[string]interface{})
	_ = json.Unmarshal(data, &contents)
	return &contents, nil
}
