package ekssetup

import (
	_ "bytes"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	_ "net/http"
	"regexp"
	"time"

	//"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
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
		PrivateAccess		*bool     `yaml:"PrivateAccess"`
		PublicAccess 		*bool	  `yaml:"PublicAccess"`
		PublicCIDR 		 interface{}     `yaml:"PublicCIDR"`
		Logging          interface{} `yaml:"Logging"`
		KMSKey			 string   	`yaml:"KMSKey"`
		Tags 		     interface{} `yaml:"Tags"`
		KubernetesNetworkCIDR string `yaml:"KubernetesNetworkCIDR"`
	} `yaml:"Master"`
}
type EksMasterSDK struct {
	Master struct {
		KubernetesVersion string      `yaml:"KubernetesVersion"`
		SecurityGroupIds  string      `yaml:"SecurityGroupIds"`
		SubnetIds         []*string   `yaml:"SubnetIds"`
		PrivateAccess		*bool     `yaml:"PrivateAccess"`
		PublicAccess 		*bool	  `yaml:"PublicAccess"`
		PublicCIDR 		[]*string     `yaml:"PublicCIDR"`
		Logging         []*string `yaml:"Logging"`
		KMSKey			*string   	`yaml:"KMSKey"`
		Tags 		  map[string]string `yaml:"Tags"`
		KubernetesNetworkCIDR string `yaml:"KubernetesNetworkCIDR"`
	} `yaml:"Master"`
}
type NodeList struct {
	Nodes []Nodevalues `yaml:"Nodes"`
}
type Nodevalues struct {
	NodegroupName string      		`yaml:"NodegroupName"`
	InstanceTypes []*string     	`yaml:"InstanceTypes"`
	SubnetIds     []*string 		`yaml:"SubnetIds"`
	SpotInstance	 bool			`yaml:"SpotInstance"`
	DiskSize 	 string			`yaml:"DiskSize"`
	Labels 		 map[string]string `yaml:"Labels"`
	AmiType 	 string           `yaml:"AmiType"`
	Tags 		  map[string]string `yaml:"Tags"`
	ScalingConfig map[string]int `yaml:"ScalingConfig"`
	RemoteAccess  RemoteAccess `yaml:"RemoteAccess"`
	Taints 		  []Taints	`yaml:"Taints"`
}
type RemoteAccess struct {
	SSHKey string `yaml:"SSHKey"`
	SourceSecurityGroups []string `yaml:"SourceSecurityGroups"`
}
type Taints struct {
	Effect string `yaml:"Effect"`
	Key string `yaml:"Key"`
	Value string `yaml:"Value"`
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create Directory
	if _, err := os.Stat("templates"); os.IsNotExist(err) {
		os.Mkdir("templates", 0775)
	}

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
func getFileFromURL(fileName string, fileUrl string) {
	err := DownloadFile(fileName, fileUrl)
	if err != nil {
		panic(err)
	}
	fmt.Println("Downloaded: " + fileUrl)

}
func AddFileToS3(sess *session.Session, VPC []byte, Nodes []byte, s3 string, cluster string) (error, string) {
	svc := s3manager.NewUploader(sess)

	// Open the file for use
	VPCfile := string(VPC)
	VPCfileName := cluster + "-VPC" + ".yml"
	println("VPC Cloudformation YAML Name: \n", VPCfileName)
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
		return err, ""
	}

	if err != nil {
		// Do your error handling here
		return err, ""
	}
	_, err = svc.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3),              // Bucket to be used
		Key:    aws.String(NodesfileName),   // Name of the file to be saved
		Body:   strings.NewReader(Nodefile), // File
	})
	if err != nil {
		// Do your error handling here
		return err, ""
	}

	return err, VPCfileName
}

//Setup EKS Cluster

func ReadEKSYaml(f []byte) {
	////Setting up variables
	ElementsSubnetIDs := make(map[string]string)

	var MClusterName, vpcsecuritygps, vpcclustername, Profile, Acceesskey, Secretkey, Region, Cluster, VPCfileName, EksfileName, VPCSourceFile string
	var nodelen int
	var vpcsubnets *string
	var MSubnetIds []*string

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

	fmt.Println("Creating sessions.......")

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
	fmt.Println("Session created \n")

	////Setting up S3 Bucket
	fmt.Println("Setting up S3 bucket \n")
	S3Name := eksSession.Cloud.Bucket

	//Loading Yaml
	VPCFile, err := ioutil.ReadFile("templates/0001-vpc.yaml")
	NodeFile, err := ioutil.ReadFile("templates/0007-esk-managed-node-group.yaml")

	//Add Yaml templates to s3
	err, VPCfileName = AddFileToS3(sess, VPCFile, NodeFile, S3Name, Cluster)
	if err != nil {
		log.Fatal(err)
	}

	////Checking if VPC is enabled
	fmt.Println("Checking if VPC creation enabled")
	err = yaml.Unmarshal([]byte(file), &eksvpc)
	if err != nil {
		panic(err)
	}
	VPCName := eksvpc.VPC.VpcBlock
	PublicSubnetLen := len(eksvpc.VPC.PublicSubnets)
	PrivateSubnetLen := len(eksvpc.VPC.PrivateSubnets)

	if PublicSubnetLen != PrivateSubnetLen {
		fmt.Println("PublicSubnets and PrivateSubnets count should  be same\n")
		os.Exit(255)
	}

	if PublicSubnetLen == 2 {
		VPCSourceFile = "https://k8s-cloud-templates.s3.amazonaws.com/vpc-4subnets.yaml"
	} else if PublicSubnetLen == 3 {
		VPCSourceFile = "https://k8s-cloud-templates.s3.amazonaws.com/vpc-6subnets.yaml"
	} else {
		fmt.Println("Only 2 or 4 Public/Private Subnet pairs are accepted")
		os.Exit(255)
	}

	getFileFromURL("templates/0001-vpc.yaml", VPCSourceFile)
	getFileFromURL("templates/0007-esk-managed-node-group.yaml", "https://k8s-cloud-templates.s3.amazonaws.com/0007-esk-managed-node-group.yaml")

	if VPCName != "" {
		fmt.Println("VPC creation enabled, checking VPC state.......\n")
		vpcsubnets, vpcsecuritygps, vpcclustername, ElementsSubnetIDs = Create_VPC(sess, file, Cluster, S3Name, VPCfileName)
	} else {
		fmt.Println("VPC creation Not Enabled\n")
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

	Master := eksMaster
	if (EksMaster{}) != Master  {
		fmt.Println("Master creation enabled, creating/updating stacks.......")
		//MClusterName, MSubnetIds = Create_Master(sess, vpcsecuritygps, vpcclustername, vpcsubnets, ElementsSubnetIDs, file, Cluster, S3Name, EksfileName)
		MClusterName, MSubnetIds, _ = Create_Master_sdk(sess, vpcsecuritygps, vpcclustername, vpcsubnets, ElementsSubnetIDs, file, Cluster, S3Name, EksfileName)

		if nodelen == 0 {
			fmt.Println("Master creation completed, no node groups provided.......")
		} else if nodelen != 0 {
			fmt.Println("Master creation completed, node groups listed.......")
			fmt.Println("Creating node groups.......")
			for i := 0; i < nodelen; i++ {
				println("Subnets available in Master: ", awsutil.StringValue(MSubnetIds))
				//println("ClusterName  from Master: ", MClusterName)
				//Create_Node(sess, i, MClusterName, MSubnetIds, ElementsSubnetIDs, file, Cluster, S3Name, NodesfileName)
				Create_NodeGroup_SDK(sess, i, MClusterName, MSubnetIds, ElementsSubnetIDs, file, Cluster)

			}
		}
	} else {
		fmt.Println("EKS Cluster Not Enabled")
	}

}
func Create_VPC(sess *session.Session, file []byte, cluster string, S3 string, VPCFilename string) (*string, string, string, map[string]string) {

	// Creating vars
	fileVPC := file
	var eksvpc EksVPC

	ElementsSubnetIDs := make(map[string]string)
	ElementsCreate := make(map[string]string)
	ElementsUpdate := make(map[string]string)
	ElementsSubnets := make(map[string]string)

	var v = cftvpc{}
	var value, Keyname string
	var vpcsubnets *string
	var vpcsecuritygps string
	var vpcclustername string
	err := yaml.Unmarshal([]byte(fileVPC), &eksvpc)

	//StackName := eksvpc.VPC.StackName
	StackName := cluster + "-VPC-Stack"
	VpcBlock := eksvpc.VPC.VpcBlock
	ClusterName := cluster
	//ClusterName := eksvpc.VPC.ClusterName

	Module := "VPC"
	ElementsCreate = map[string]string{
		"VpcBlock":    VpcBlock,
		"ClusterName": ClusterName,
	}
	// specify elements that needs to be updated below as above
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
		fmt.Println("Keyname: ", PublicSubnetKeys[i])
		value, _ = strconv.Unquote(awsutil.StringValue(PublicSubnet[Keyname]))
		fmt.Println("Values: ", value)
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
		//ElementsUpdate[Keyname] = value // Commenting this as to not up update VPC after it got created
	}

	//TemplateURL, _ := yaml.Get("VPC").Get("TemplateURL").String()
	//TemplateURL := eksvpc.VPC.TemplateURL
	TemplateURL := "https://" + S3 + ".s3.amazonaws.com/" + VPCFilename
	v.StackName = StackName
	v.TemplateURL = TemplateURL

	//Passing values for creating stack

	//	fmt.Println(".......ElementsCreate.....", ElementsCreate)
	//Passing values for updating Stack

	fmt.Println("StackName: ", v.StackName)
	fmt.Println("TemplateURL: ", v.TemplateURL)
	//fmt.Println("Parameters to be created", ElementsCreate)
	//fmt.Println("Parameters to be updated", ElementsUpdate)

	if err != nil {
		fmt.Println(os.Stderr, "YAML Prasing failed with Error: ", err)
		os.Exit(1)
	}

	// Calling stack validation

	a, b := ValidateStack(sess, v.TemplateURL, ElementsCreate, ElementsUpdate)

	// Calling outputs from created/updated stack

	ListStack(sess, v, a, b, Module)

	NoOP := len(CheckStack(sess, StackName).Stacks[0].Outputs)

	for p := 0; p < NoOP; p++ {
	//	time.Sleep(2 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "SubnetIds" {
		//	time.Sleep(2 * time.Second)
			value := CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue
			fmt.Println("Subnets: ", awsutil.StringValue(value))
			vpcsubnets = value
		//	time.Sleep(2 * time.Second)
			//fmt.Println(awsutil.StringValue(vpcsubnets))
		}
	}
	for p := 0; p < NoOP; p++ {
	//	time.Sleep(2 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "SecurityGroups" {
	//		time.Sleep(2 * time.Second)
			value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
			fmt.Println("SecurityGroups: ", value)
			vpcsecuritygps = value
	//		time.Sleep(2 * time.Second)
		}
	}
	for p := 0; p < NoOP; p++ {
	//	time.Sleep(2 * time.Second)
		k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
		var c string = strings.Trim(k, "\"")
		if string(c) == "ClusterName" {
		//	time.Sleep(2 * time.Second)
			value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
			fmt.Println("Cluster Name: ", value)
			vpcclustername = value
		//	time.Sleep(2 * time.Second)
		}
	}

	// Creating SubnetIDs elements

	for i := 0; i < NoofKeysprivate; i++ {
		Keyname = PrivateSubnetKeys[i]
		for p := 0; p < NoOP; p++ {
			//time.Sleep(2 * time.Second)
			k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
			var c string = strings.Trim(k, "\"")
			if string(c) == Keyname {
				//time.Sleep(2 * time.Second)
				value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
				//fmt.Printf(Keyname, ":", value)
				fmt.Printf("%v", Keyname)
				fmt.Printf(":")
				fmt.Printf("%v\n", value)
				ElementsSubnetIDs[strconv.Quote(Keyname)] = value
				//time.Sleep(2 * time.Second)
			}
		}
	}
	for i := 0; i < NoofKeyspublic; i++ {
		Keyname = PublicSubnetKeys[i]
		for p := 0; p < NoOP; p++ {
			//time.Sleep(2 * time.Second)
			k := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputKey)
			var c string = strings.Trim(k, "\"")
			if string(c) == Keyname {
				//time.Sleep(2 * time.Second)
				value := awsutil.StringValue(CheckStack(sess, StackName).Stacks[0].Outputs[p].OutputValue)
				fmt.Printf("%v", Keyname)
				fmt.Printf(":")
				fmt.Printf("%v\n", value)
				ElementsSubnetIDs[strconv.Quote(Keyname)] = value
				//time.Sleep(2 * time.Second)
			}
		}
	}

	//	fmt.Printf("ElementsSubnetIDs: %v\n", ElementsSubnetIDs)
	list := CheckStack(sess, StackName).Stacks[0].StackName
	fmt.Println("StackID of the Stack: ", awsutil.StringValue(list))
	if err != nil {
		panic(err)
	}

	return vpcsubnets, vpcsecuritygps, vpcclustername, ElementsSubnetIDs
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
		fmt.Println(os.Stderr, "Validation Failed with Error: ", err)
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
func ListStack(sess *session.Session, c cftvpc, stackcreate []*awscf.Parameter, stackupdate []*awscf.Parameter, Module string) {

	type ByAge []Config
	var v = c
	var count = 0
	svc := awscf.New(sess)
	params := &awscf.DescribeStacksInput{}
	resp, err := svc.DescribeStacks(params)

	if err != nil {
		fmt.Println(os.Stderr, "Validation Failed with Error: ", err)
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
				////time.Sleep(2 * time.Second)
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
		//fmt.Println(present)
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
					if Module == "VPC" {
						fmt.Println("VPC already created and cannot be updated")
						//UpdateStack(sess, v, stackupdate)
						for {
							var a string = "UPDATE_IN_PROGRESS"
							b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
							var c string = strings.Trim(b, "\"")
							if a != c {
								fmt.Println("Status: ", b)
								////time.Sleep(2 * time.Second)
								fmt.Println()
								break
							}
						}
						println("Checking Status.......")
						//fmt.Println("Update Completed")
					} else {
						UpdateStack(sess, v, stackupdate)
						println("Checking Status.......")
						for {
							var a string = "UPDATE_IN_PROGRESS"
							b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
							var c string = strings.Trim(b, "\"")
							if a != c {
								fmt.Println("Status: ", b)
								////time.Sleep(2 * time.Second)
								fmt.Println()
								break
							}
						}
						fmt.Println("Update Completed")
					}

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
					//time.Sleep(2 * time.Second)
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
		fmt.Println(os.Stderr, "Listing failed with Error: ", err)
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
		fmt.Println(os.Stderr, "Creation Failed with Error: ", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Println("Stack Creation passed")
	}

	//fmt.Printf(awsutil.StringValue(rep))

	return rep
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
		exitErrorf("unknown error occurred: ", err)
	}

	//fmt.Printf("StackID: ", awsutil.StringValue(resp.StackId))
	return resp

}
func Create_Master_sdk(sess *session.Session, vpcsecuritygps string, vpcclustername string, vpcsubnets *string, ElementsSubnetIDs map[string]string, file []byte, cluster string, s3 string, eksfileName string) (string, []*string, string) {

	// Creating vars
	svc := eks.New(sess)
	var ClusterName, SecurityGroupIds, KubernetesNetworkCIDR string
	var SubnetIds []*string
	var eksmaster EksMasterSDK
	var Tags map[string]string
	FileMaster := file
	err := yaml.Unmarshal([]byte(FileMaster), &eksmaster)
	Module := "Master"


	KubernetesVersion := eksmaster.Master.KubernetesVersion
	Tags = eksmaster.Master.Tags
	KubernetesNetworkCIDR = eksmaster.Master.KubernetesNetworkCIDR
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
	if vpcsubnets == nil {
		SubnetIds = eksmaster.Master.SubnetIds
	} else if vpcsubnets != nil {
		arrayl := eksmaster.Master.SubnetIds
		//		fmt.Printf("Check Arryl ...\n", arrayl)
		arrlen := len(arrayl)
		//		fmt.Printf("Array Lenght...\n", arrlen)
		arropt := make([]string, int(arrlen))
		if arrlen == 0 {

			vpcsubnetssplit := strings.Split(awsutil.StringValue(vpcsubnets), ",")
			pvpcsubnets := []*string{}
			for i := range awsutil.StringValue(vpcsubnetssplit) {
				pvpcsubnets = append(pvpcsubnets, &vpcsubnetssplit[i])
			}
			SubnetIds = pvpcsubnets

		} else if arrlen != 0 {
			for i := 0; i < arrlen; i++ {
				var subnetIDValue string
				subnetName := strings.TrimSpace(awsutil.StringValue(arrayl[i]))
				b := strings.TrimSpace(strconv.Quote(strings.Trim(subnetName, "\"")))
				if ElementsSubnetIDs[b] != "" {
					subnetIDValue = strings.TrimSpace(ElementsSubnetIDs[b])
				} else if ElementsSubnetIDs[b] == "" {
					subnetIDValue = strings.TrimSpace(string(b))
				}
				//fmt.Println(subnetIDValue)
				arropt[i] = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(subnetIDValue, "\\", ""), "\"", ""))
			}

			pvpcsubnets := []*string{}
			for i := range arropt {
				pvpcsubnets = append(pvpcsubnets, &arropt[i])
			}
			SubnetIds = pvpcsubnets
		}
	}
	tags2 := map[string]*string{}
	for key, _ := range Tags {
		value := Tags[key]
		tags2[key] = &value
	}

	fmt.Println("ClusterName: ", ClusterName)
	fmt.Println("SecurityGroups: ", SecurityGroupIds)
	fmt.Println("Subnets: ", awsutil.StringValue(SubnetIds))
	fmt.Println("Tags: ", awsutil.StringValue(tags2))

	if err != nil {
		fmt.Println(os.Stderr, "Prasing failed with Error: ", err)
		os.Exit(1)
	}

    loglice := []string{"api","scheduler", "authenticator", "audit", "controllerManager"}
	pvplogging := []*string{}
	for i := range loglice {
		pvplogging = append(pvplogging, &loglice[i])
	}

	slice2 := pvplogging
	slice1 := eksmaster.Master.Logging
	diff2 := []*string{}
	val := difference(slice1, slice2)
	for i := range val {
		diff2 = append(diff2, &val[i])
	}

	fmt.Printf("Checking Cluster state\n")
	var status string
	status, _, _, err, _, _ = ListStack_sdk(sess, ClusterName, "",Module)
	if err != nil {
		fmt.Println(os.Stderr, "Cluster info not available - Error: ")
		//os.Exit(1)
		status = ""
	}
	fmt.Println("Status:", status)

	if status == "" {

		fmt.Println("Cluster does't exist, creating cluster")
		fmt.Println("Creating Cluster Role")
		arn := create_role(sess, ClusterName)
		fmt.Println("ARN: ", arn)

		if eksmaster.Master.Tags == nil {

			if eksmaster.Master.KMSKey == nil {

				if eksmaster.Master.Logging != nil {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						KubernetesNetworkConfig: &eks.KubernetesNetworkConfigRequest{ServiceIpv4Cidr: &KubernetesNetworkCIDR},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}

				} else {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						//EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				}


			} else {

				if eksmaster.Master.Logging != nil {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				} else {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				}

			}

		} else {

			if eksmaster.Master.KMSKey == nil {

				if eksmaster.Master.Logging != nil {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						KubernetesNetworkConfig: &eks.KubernetesNetworkConfigRequest{ServiceIpv4Cidr: &KubernetesNetworkCIDR},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
						Tags: tags2,
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}

				} else {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						//EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Tags: tags2,
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				}


			} else {

				if eksmaster.Master.Logging != nil {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
						Tags: tags2,
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				} else {
					input := &eks.CreateClusterInput{
						ClientRequestToken: nil,
						Name:               aws.String(ClusterName),
						EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
						ResourcesVpcConfig: &eks.VpcConfigRequest{
							SecurityGroupIds: []*string{
								aws.String(SecurityGroupIds),
							},
							SubnetIds:             SubnetIds,
							EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
							EndpointPublicAccess:  eksmaster.Master.PublicAccess,
							PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
						},
						RoleArn: aws.String(arn),
						Version: aws.String(KubernetesVersion),
						Tags: tags2,
					}
					result, err := svc.CreateCluster(input)
					if err != nil {
						if aerr, ok := err.(awserr.Error); ok {
							switch aerr.Code() {
							case eks.ErrCodeResourceInUseException:
								fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
							case eks.ErrCodeResourceLimitExceededException:
								fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
							case eks.ErrCodeInvalidParameterException:
								fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
							case eks.ErrCodeClientException:
								fmt.Println(eks.ErrCodeClientException, aerr.Error())
							case eks.ErrCodeServerException:
								fmt.Println(eks.ErrCodeServerException, aerr.Error())
							case eks.ErrCodeServiceUnavailableException:
								fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
							case eks.ErrCodeUnsupportedAvailabilityZoneException:
								fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
							default:
								fmt.Println(aerr.Error())
							}
						} else {
							// Print the error, cast err to awserr.Error to get the Code and
							// Message from an error.
							fmt.Println(err.Error())
						}
					}
					fmt.Println(result)
					println("Checking Status.......")
					for {
						var a string = "CREATING"
						b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
						fmt.Println("Creating Cluster...")
						if a != b {
							fmt.Println("Status: ", awsutil.StringValue(b))
							////time.Sleep(2 * time.Second)
							fmt.Println()
							break
						}
					}
				}

			}

		}

	} else if status == "ACTIVE" {

		if eksmaster.Master.KMSKey == nil {} else {
			fmt.Println("Cluster exist, updating KMS Key")
			fmt.Println("Updating encryption configs")
			input0 := &eks.AssociateEncryptionConfigInput{
				ClientRequestToken: nil,
				ClusterName:        aws.String(ClusterName),
				EncryptionConfig: []*eks.EncryptionConfig{&eks.EncryptionConfig{Resources: []*string{aws.String("secrets")}, Provider: &eks.Provider{KeyArn: eksmaster.Master.KMSKey}}},
			}
			result0, err := svc.AssociateEncryptionConfig(input0)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case eks.ErrCodeResourceInUseException:
						fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
					case eks.ErrCodeResourceLimitExceededException:
						fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
					case eks.ErrCodeInvalidParameterException:
						fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
					case eks.ErrCodeClientException:
						fmt.Println(eks.ErrCodeClientException, aerr.Error())
					case eks.ErrCodeServerException:
						fmt.Println(eks.ErrCodeServerException, aerr.Error())
					case eks.ErrCodeServiceUnavailableException:
						fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
					case eks.ErrCodeUnsupportedAvailabilityZoneException:
						fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
			}
			fmt.Println(result0)
			println("Checking Status.......")
			for {
				var a  = "UPDATING"
				b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName,"", Module)
				fmt.Println("Updating Cluster  ...")
				if a != b {
					fmt.Println("Status: ", awsutil.StringValue(b))
					time.Sleep(2 * time.Second)
					fmt.Println()
					break
				}
			}
		}

		fmt.Println("Updating cluster configs")
		input := &eks.UpdateClusterConfigInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			ResourcesVpcConfig: &eks.VpcConfigRequest{
				EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
				EndpointPublicAccess:  eksmaster.Master.PublicAccess,
				PublicAccessCidrs:     eksmaster.Master.PublicCIDR,
			},
		}
		result, err := svc.UpdateClusterConfig(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result)
		println("Checking Status.......")
		for {
			var a  = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}


		fmt.Println("Updating logging configs")
		input3 := &eks.UpdateClusterConfigInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
		}
		result3, err := svc.UpdateClusterConfig(input3)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result3)
		println("Checking Status.......")
		for {
			var a  = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

		fmt.Println("Updating cluster version")
		input2 := &eks.UpdateClusterVersionInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			Version: aws.String(KubernetesVersion),
		}
		result2, err := svc.UpdateClusterVersion(input2)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result2)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName,"", Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "UPDATING" {
		println("Checking Status.......")
		for {
			var a  = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName,"", Module)
			fmt.Println("Cluster updating ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "FAILED" {

		fmt.Println("Cluster exist, updating cluster")
		fmt.Println("Updating cluster configs")
		input := &eks.UpdateClusterConfigInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			ResourcesVpcConfig: &eks.VpcConfigRequest{
				//SecurityGroupIds: []*string{
				//	aws.String(SecurityGroupIds),
				//},
				//SubnetIds: SubnetIds,
				EndpointPrivateAccess: eksmaster.Master.PrivateAccess,
				EndpointPublicAccess: eksmaster.Master.PublicAccess,
				PublicAccessCidrs: eksmaster.Master.PublicCIDR,
			},
		}
		result, err := svc.UpdateClusterConfig(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}


		fmt.Println("Updating logging configs")
		input3 := &eks.UpdateClusterConfigInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			Logging: &eks.Logging{ClusterLogging: []*eks.LogSetup{&eks.LogSetup{Enabled: newTrue(), Types: eksmaster.Master.Logging},{Enabled: newFalse(), Types: diff2 }},},
		}
		result3, err := svc.UpdateClusterConfig(input3)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result3)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}


		fmt.Println("Updating cluster version")
		input2 := &eks.UpdateClusterVersionInput{
			ClientRequestToken: nil,
			Name:               aws.String(ClusterName),
			Version: aws.String(KubernetesVersion),
		}
		result2, err := svc.UpdateClusterVersion(input2)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result2)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName,"", Module)
			fmt.Println("Updating Cluster  ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "DELETING" {

		//status = ListStack_sdk(sess, ClusterName)
		println("Checking Status.......")
		for {
			var a string = "DELETING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
			fmt.Println("Deleting Cluster ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "CREATING" {

		//status = ListStack_sdk(sess, ClusterName)
		println("Checking Status.......")
		for {
			var a string = "CREATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, ClusterName,"", Module)
			fmt.Println("Creating Cluster ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else {
		fmt.Println(status, "check inputs")
	}
	_, results, _, _, _, _ := ListStack_sdk(sess, ClusterName, "",Module)
	fmt.Println(results)

	return ClusterName, SubnetIds, Module

}
func Create_NodeGroup_SDK(sess *session.Session, nodelen int, MClusterName string, MSubnetIds []*string, ElementsSubnetIDs map[string]string, f []byte, cluster string) {

	svc := eks.New(sess)

	// Creating vars
	var RemoteAccessSSH, AmiType, NodeClusterName,  capacitytype, disksize, NodeRole, NodegroupName, ReleaseVersion, Version string
	var Tags, Labels map[string]string
	var ScalingConfig map[string]int
	var InstanceTypes []*string
	var RemoteAccessSG []string
	var SpotInstance bool
	var NewDiskLimit, MinSize,MaxSize,DesiredSize  int
	var ConfNode NodeList
	var NSubnetIds []*string
	//var Taints map[string]map[string]string

	file := f
	err := yaml.Unmarshal([]byte(file), &ConfNode)
	if err != nil {
		err.Error()
		panic(err)
	}
	//Module := "Nodes"

	err = yaml.Unmarshal([]byte(file), &ConfNode)
	if err != nil {
		panic(err)
	}


	Module := "NodeGroup"
	NodegroupName = ConfNode.Nodes[int(nodelen)].NodegroupName
	InstanceTypes = ConfNode.Nodes[int(nodelen)].InstanceTypes
	SpotInstance = ConfNode.Nodes[int(nodelen)].SpotInstance
	disksize = ConfNode.Nodes[int(nodelen)].DiskSize
	Labels = ConfNode.Nodes[int(nodelen)].Labels
	AmiType = ConfNode.Nodes[int(nodelen)].AmiType
	Tags = ConfNode.Nodes[int(nodelen)].Tags
	ScalingConfig = ConfNode.Nodes[int(nodelen)].ScalingConfig
	RemoteAccessSG = ConfNode.Nodes[int(nodelen)].RemoteAccess.SourceSecurityGroups
	RemoteAccessSSH = ConfNode.Nodes[int(nodelen)].RemoteAccess.SSHKey
	TaintsTotal := []*eks.Taint{}
    for i := range ConfNode.Nodes[int(nodelen)].Taints{
		m := eks.Taint{
			Effect: aws.String(ConfNode.Nodes[int(nodelen)].Taints[i].Effect),
			Key:    aws.String(ConfNode.Nodes[int(nodelen)].Taints[i].Key),
			Value:  aws.String(ConfNode.Nodes[int(nodelen)].Taints[i].Value),
		}
		TaintsTotal = append(TaintsTotal,&m)
	}

	if ScalingConfig["DesiredSize"] == 0 {
		DesiredSize = 1
	} else {
		DesiredSize = ScalingConfig["DesiredSize"]
	}
	if ScalingConfig["MaxSize"] == 0 {
		MaxSize = 1
	} else {
		MaxSize = ScalingConfig["MaxSize"]
	}
	if ScalingConfig["MinSize"] == 0 {
		MinSize = 1
	} else {
		MinSize = ScalingConfig["MinSize"]
	}

	labels2 := map[string]*string{}
	for key, _ := range Labels {
		value := Labels[key]
		labels2[key] = &value
	}

	tags2 := map[string]*string{}
	for key, _ := range Tags {
		value := Tags[key]
		tags2[key] = &value
	}

	if MClusterName == "" {
		NodeClusterName = cluster
	} else if MClusterName != "" {
		NodeClusterName = MClusterName
	}
	if InstanceTypes == nil {
		//NodeClusterName = ConfNode.Nodes[int(nodelen)].ClusterName
//		val := "t3.large"
//		InstanceTypes[0] = &val
	}
	if SpotInstance == true {
		capacitytype = "SPOT"
	} else {
		capacitytype = "ON_DEMAND"
	}
	if disksize != "" {
		errcatch := err
		NewDiskLimit, errcatch = strconv.Atoi(disksize)
		if errcatch != nil {
			result3, _ := regexp.MatchString("g", strings.ToLower(disksize))
			if result3 == true{
				NewDiskLimit, _ = strconv.Atoi(strings.Trim(strings.ToLower(disksize), "gb"))
			} else {
				result4, _ := regexp.MatchString("gb", strings.ToLower(disksize))
				if result4 == true {
					NewDiskLimit, _ = strconv.Atoi(strings.Trim(strings.ToLower(disksize), "gb"))
				} else {
					fmt.Println("Please provide valid disk size in GB; units GB or G")
				}
			}
		}
	} else {
		NewDiskLimit = 20
	}
	if MSubnetIds == nil {
		NSubnetIds = ConfNode.Nodes[int(nodelen)].SubnetIds
	} else if MSubnetIds != nil {
		arrayl := ConfNode.Nodes[int(nodelen)].SubnetIds
		fmt.Println("Subnets passed form Cluster YML: ", awsutil.StringValue(arrayl))
		arrlen := len(arrayl)
		arropt := make([]string, int(arrlen))
		if arrlen == 0 {
			NSubnetIds = MSubnetIds
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
				arropt[i] = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(subnetIDValue, "\\", ""), "\"", ""))
				//arropt[i] = subnetIDValue
			}
			pvpcsubnets := []*string{}
			for i := range arropt {
				pvpcsubnets = append(pvpcsubnets, &arropt[i])
			}
			NSubnetIds = pvpcsubnets
			//fmt.Println(awsutil.StringValue(NSubnetIds))
			//NSubnetIds, _ = strconv.Unquote(awsutil.StringValue(strings.Join(arropt, ",")))
		}
	}

	NodeRole, NodeRoleName := get_role(sess, NodeClusterName)

	if err != nil {
		fmt.Println(os.Stderr, "YAML Prasing failed with Error: ", err)
		os.Exit(1)
	}

	fmt.Println("Node Cluster Name: ", NodeClusterName)
	fmt.Println("Node Subnets: ", MSubnetIds)
	fmt.Println("Node Group Name: ", NodegroupName)
	fmt.Println("Node SubnetIds: ", NSubnetIds)
	fmt.Println("Node Taints: ",awsutil.StringValue(TaintsTotal))
	fmt.Println("Node Tags: ", awsutil.StringValue(Tags))
	fmt.Println("Node Labels: ", awsutil.StringValue(labels2))
	fmt.Println("Node MaxSize: ", MaxSize)
	fmt.Println("Node MinSize: ", MinSize)
	fmt.Println("Node DesiredSize: ", DesiredSize)
	fmt.Println("Node InstanceType: ", InstanceTypes)
	fmt.Println("Node SSHKeyName: ", RemoteAccessSSH)
	fmt.Println("Node SSHSecuriyGroup: ", RemoteAccessSG)
	fmt.Println("Node AMIType: ", AmiType)
	fmt.Println("Node DiskSizeGB: ", NewDiskLimit)
	fmt.Println("Node CapacityType: ", capacitytype)

	update_noderole(sess, NodeRoleName)

	fmt.Printf("Checking Node state\n")
	var status string
	status, _, _, err, existinglabels, existingTaints := ListStack_sdk(sess, cluster, NodegroupName, Module)
	if err != nil {
		fmt.Println(os.Stderr, "NodeGroup info not available - Error: ", err)
		//os.Exit(1)
		status = ""
	}
	fmt.Println("Status:", status)

	if status == "" {
		fmt.Println("NodeGroup doesn't exit, creating NodeGroup")
		if tags2 == nil {
			update_noderole(sess, NodeRoleName)
			input := &eks.CreateNodegroupInput{
				AmiType: &AmiType,
				CapacityType: &capacitytype,
				ClientRequestToken: nil,
				ClusterName: &NodeClusterName,
				DiskSize: aws.Int64(int64(NewDiskLimit)),
				InstanceTypes: InstanceTypes,
				Labels: labels2,
				NodeRole: &NodeRole,
				NodegroupName: &NodegroupName,
				Subnets: NSubnetIds,
				ReleaseVersion: &ReleaseVersion,
				ScalingConfig: &eks.NodegroupScalingConfig{
					DesiredSize: aws.Int64(int64(DesiredSize)),
					MaxSize:     aws.Int64(int64(MaxSize)),
					MinSize:     aws.Int64(int64(MinSize)),
				},
				RemoteAccess: &eks.RemoteAccessConfig{
					Ec2SshKey:            &RemoteAccessSSH,
					SourceSecurityGroups: aws.StringSlice(RemoteAccessSG),
				},
				Version: &Version,
				//Taints: []*eks.Taint{&eks.Taint{
				//	Effect: nil,
				//	Key:    nil,
				//	Value:  nil,
				//}},
				Taints: TaintsTotal,
				//LaunchTemplate: "",
			}
			result, err := svc.CreateNodegroup(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case eks.ErrCodeResourceInUseException:
						fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
					case eks.ErrCodeResourceLimitExceededException:
						fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
					case eks.ErrCodeInvalidParameterException:
						fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
					case eks.ErrCodeClientException:
						fmt.Println(eks.ErrCodeClientException, aerr.Error())
					case eks.ErrCodeServerException:
						fmt.Println(eks.ErrCodeServerException, aerr.Error())
					case eks.ErrCodeServiceUnavailableException:
						fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
					case eks.ErrCodeUnsupportedAvailabilityZoneException:
						fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
			}
			fmt.Println(result)
			println("Checking Status.......")
			for {
				var a string = "CREATING"
				b, _, _, _, _, _ := ListStack_sdk(sess, NodeClusterName, NodegroupName,Module)
				fmt.Println("Creating NodeGroups...")
				if a != b {
					fmt.Println("Status: ", awsutil.StringValue(b))
					////time.Sleep(2 * time.Second)
					fmt.Println()
					break
				}
			}
		} else {
			input := &eks.CreateNodegroupInput{
				AmiType: &AmiType,
				CapacityType: &capacitytype,
				ClientRequestToken: nil,
				ClusterName: &NodeClusterName,
				DiskSize: aws.Int64(int64(NewDiskLimit)),
				InstanceTypes: InstanceTypes,
				Labels: labels2,
				NodeRole: &NodeRole,
				NodegroupName: &NodegroupName,
				Subnets: NSubnetIds,
				ReleaseVersion: &ReleaseVersion,
				Tags: tags2,
				Version: &Version,
				ScalingConfig: &eks.NodegroupScalingConfig{
					DesiredSize: aws.Int64(int64(DesiredSize)),
					MaxSize:     aws.Int64(int64(MaxSize)),
					MinSize:     aws.Int64(int64(MinSize)),
				},
				RemoteAccess: &eks.RemoteAccessConfig{
					Ec2SshKey:            &RemoteAccessSSH,
					SourceSecurityGroups: aws.StringSlice(RemoteAccessSG),
				},
				Taints: TaintsTotal,
				//Taints: []*eks.Taint{&eks.Taint{
				//	Effect: nil,
				//	Key:    nil,
				//	Value:  nil,
				//},&eks.Taint{
				//	Effect: nil,
				//	Key:    nil,
				//	Value:  nil,
				//}},
				//LaunchTemplate: "",
			}
			result, err := svc.CreateNodegroup(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case eks.ErrCodeResourceInUseException:
						fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
					case eks.ErrCodeResourceLimitExceededException:
						fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
					case eks.ErrCodeInvalidParameterException:
						fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
					case eks.ErrCodeClientException:
						fmt.Println(eks.ErrCodeClientException, aerr.Error())
					case eks.ErrCodeServerException:
						fmt.Println(eks.ErrCodeServerException, aerr.Error())
					case eks.ErrCodeServiceUnavailableException:
						fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
					case eks.ErrCodeUnsupportedAvailabilityZoneException:
						fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
			}
			fmt.Println(result)
			println("Checking Status.......")
			for {
				var a string = "CREATING"
				b, _, _, _, _, _ := ListStack_sdk(sess, NodeClusterName, NodegroupName,Module)
				fmt.Println("Creating NodeGroups...")
				if a != b {
					fmt.Println("Status: ", awsutil.StringValue(b))
					////time.Sleep(2 * time.Second)
					fmt.Println()
					break
				}
			}
		}

	} else if status == "ACTIVE" {

		println("Updating Node Groups.......")

		//s2>s1
		slice2 := len(existinglabels)
		slice1 := len(labels2)
		diff2 := []*string{}
		if slice2 > slice1 {
			//pkeysslice2 :=
			keysslice2 := make([]*string, slice2)
			i := 0
			for k := range existinglabels {
				keysslice2[i] = &k
				i++
			}
			//pkeysslice1
			keysslice1 := make([]*string, slice1)
			j := 0
			for k := range labels2 {
				keysslice1[j] = &k
				j++
			}
			val := difference(keysslice2, keysslice1)
			for i := range val {
				diff2 = append(diff2, &val[i])
			}
		} else {
			diff2 = nil
		}
		fmt.Println("Labels to be removed: ", awsutil.StringValue(diff2))

		TaintsTotalRemoved := []*eks.Taint{}
		taints1 := len(existingTaints)
		taints2 := len(TaintsTotal)
		fmt.Println(awsutil.StringValue(existingTaints), awsutil.StringValue(TaintsTotal))
		//fmt.Println("t1:", taints1, "t2:", taints2 )
		if taints1 > taints2 {
			i := 0
			for _ = range existingTaints {
				valstaints1 := existingTaints[i].Value
				keystaints1 := existingTaints[i].Key
				effstaints1 := existingTaints[i].Effect
				i++
				j := 0
				for _ = range TaintsTotal {
					valstaints2 := TaintsTotal[i].Value
					keystaints2 := TaintsTotal[i].Key
					effstaints2 := TaintsTotal[i].Effect
					if valstaints1 == valstaints2 || keystaints1 == keystaints2 || effstaints1 == effstaints2 {
						fmt.Println("No Action needed, Taint either has to be updated or felt the same")
						fmt.Println(awsutil.StringValue(keystaints1), awsutil.StringValue(keystaints2))
						fmt.Println(awsutil.StringValue(valstaints1), aws.StringValue(valstaints2))
						fmt.Println(awsutil.StringValue(effstaints1), aws.StringValue(effstaints2))
					} else {
						fmt.Println("Taint needs to be deleted")
						fmt.Println(awsutil.StringValue(keystaints1), awsutil.StringValue(valstaints1), awsutil.StringValue(effstaints1))
						m := eks.Taint{
							Effect: keystaints1,
							Key:    keystaints1,
							Value:  effstaints1,
						}
						TaintsTotalRemoved = append(TaintsTotalRemoved,&m)
					}
				}
				j++
			}
		} else {
			TaintsTotalRemoved = nil
		}
		fmt.Println("Taints to be removed: ", awsutil.StringValue(TaintsTotalRemoved))

		input := &eks.UpdateNodegroupConfigInput{
			ClientRequestToken: nil,
			ClusterName: &NodeClusterName,
			NodegroupName: &NodegroupName,
			ScalingConfig: &eks.NodegroupScalingConfig{
				DesiredSize: aws.Int64(int64(DesiredSize)),
				MaxSize:     aws.Int64(int64(MaxSize)),
				MinSize:     aws.Int64(int64(MinSize)),
			},
			Labels: &eks.UpdateLabelsPayload{
				AddOrUpdateLabels: labels2,
				RemoveLabels:      diff2,
			},
			Taints: &eks.UpdateTaintsPayload{
				AddOrUpdateTaints: TaintsTotal,
				RemoveTaints:      TaintsTotalRemoved,
			},
		}
		result, err := svc.UpdateNodegroupConfig(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, NodeClusterName, NodegroupName,Module)
			fmt.Println("Updating Node Group ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}


	} else if status == "UPDATING" {

		println("Checking Status.......")
		for {
			var a  = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, cluster,NodegroupName, Module)
			fmt.Println("NodeGroup getting updated ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "FAILED" {

		fmt.Println("Cluster exist, updating cluster")
		fmt.Println("Updating NodeGroup configs")
		slice2 := len(existinglabels)
		slice1 := len(labels2)
		diff2 := []*string{}
		if slice2 > slice1 {
			//pkeysslice2 :=
			keysslice2 := make([]*string, slice2)
			i := 0
			for k := range existinglabels {
				keysslice2[i] = &k
				i++
			}
			//pkeysslice1
			keysslice1 := make([]*string, slice1)
			j := 0
			for k := range labels2 {
				keysslice1[j] = &k
				j++
			}
			val := difference(keysslice2, keysslice1)
			for i := range val {
				diff2 = append(diff2, &val[i])
			}
		} else {
			diff2 = nil
		}
		fmt.Println("Labels to be removed: ", awsutil.StringValue(diff2))

		TaintsTotalRemoved := []*eks.Taint{}
		taints1 := len(existingTaints)
		taints2 := len(TaintsTotal)
		fmt.Println(awsutil.StringValue(existingTaints), awsutil.StringValue(TaintsTotal))
		//fmt.Println("t1:", taints1, "t2:", taints2 )
		if taints1 > taints2 {
			i := 0
			for _ = range existingTaints {
				valstaints1 := existingTaints[i].Value
				keystaints1 := existingTaints[i].Key
				effstaints1 := existingTaints[i].Effect
				i++
				j := 0
				for _ = range TaintsTotal {
					valstaints2 := TaintsTotal[i].Value
					keystaints2 := TaintsTotal[i].Key
					effstaints2 := TaintsTotal[i].Effect
					if valstaints1 == valstaints2 || keystaints1 == keystaints2 || effstaints1 == effstaints2 {
						fmt.Println("No Action needed, Taint either has to be updated or felt the same")
						fmt.Println(awsutil.StringValue(keystaints1), awsutil.StringValue(keystaints2))
						fmt.Println(awsutil.StringValue(valstaints1), aws.StringValue(valstaints2))
						fmt.Println(awsutil.StringValue(effstaints1), aws.StringValue(effstaints2))
					} else {
						fmt.Println("Taint needs to be deleted")
						fmt.Println(awsutil.StringValue(keystaints1), awsutil.StringValue(valstaints1), awsutil.StringValue(effstaints1))
						m := eks.Taint{
							Effect: keystaints1,
							Key:    keystaints1,
							Value:  effstaints1,
						}
						TaintsTotalRemoved = append(TaintsTotalRemoved,&m)
					}
				}
				j++
			}
		} else {
			TaintsTotalRemoved = nil
		}
		fmt.Println("Taints to be removed: ", awsutil.StringValue(TaintsTotalRemoved))

		input := &eks.UpdateNodegroupConfigInput{
			ClientRequestToken: nil,
			ClusterName: &NodeClusterName,
			NodegroupName: &NodegroupName,
			ScalingConfig: &eks.NodegroupScalingConfig{
				DesiredSize: aws.Int64(int64(DesiredSize)),
				MaxSize:     aws.Int64(int64(MaxSize)),
				MinSize:     aws.Int64(int64(MinSize)),
			},
			Labels: &eks.UpdateLabelsPayload{
				AddOrUpdateLabels: labels2,
				RemoveLabels:      diff2,
			},
			Taints: &eks.UpdateTaintsPayload{
				AddOrUpdateTaints: TaintsTotal,
				RemoveTaints:      TaintsTotalRemoved,
			},
		}
		result, err := svc.UpdateNodegroupConfig(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeResourceInUseException:
					fmt.Println(eks.ErrCodeResourceInUseException, aerr.Error())
				case eks.ErrCodeResourceLimitExceededException:
					fmt.Println(eks.ErrCodeResourceLimitExceededException, aerr.Error())
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				case eks.ErrCodeUnsupportedAvailabilityZoneException:
					fmt.Println(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
		}
		fmt.Println(result)
		println("Checking Status.......")
		for {
			var a string = "UPDATING"
			b, _, _, _, _, _ := ListStack_sdk(sess, NodeClusterName, NodegroupName,Module)
			fmt.Println("Updating Node Group ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}


	} else if status == "DELETING" {

		println("Checking Status.......")
		fmt.Println("NodeGroup getting deleted, updates will not be performed ...")
		for {
			var a string = "DELETING"
			b, _, _, _, _, _ := ListStack_sdk(sess, cluster, NodegroupName,Module)
			fmt.Println("NodeGroup getting deleted ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else if status == "CREATING" {

		//status = ListStack_sdk(sess, ClusterName)
		println("Checking Status.......")
		for {
			var a string = "CREATING"
			fmt.Println("NodeGroup getting created, updates will not to performed")
			b, _, _, _, _, _ := ListStack_sdk(sess, cluster,NodegroupName, Module)
			fmt.Println("NodeGroup getting created ...")
			if a != b {
				fmt.Println("Status: ", awsutil.StringValue(b))
				////time.Sleep(2 * time.Second)
				fmt.Println()
				break
			}
		}

	} else {
		fmt.Println(status, "check inputs")
	}
	_, _, results, _, _, _ := ListStack_sdk(sess, cluster, NodegroupName,Module)
	fmt.Println(results)

}
func newTrue() *bool {
	b := true
	return &b
}
func newFalse() *bool {
	c := false
	return &c
}
func create_role(sess *session.Session, clustername string) string {

	svc := iam.New(sess)
	input := &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String("{\n  \"Version\": \"2012-10-17\",\n  \"Statement\": [\n    {\n      \"Effect\": \"Allow\",\n      \"Principal\": {\n        \"Service\": \"eks.amazonaws.com\"\n      },\n      \"Action\": \"sts:AssumeRole\"\n    }\n  ]\n}"),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(clustername),
	}
	result, err := svc.CreateRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeEntityAlreadyExistsException:
				fmt.Println(iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
			case iam.ErrCodeMalformedPolicyDocumentException:
				fmt.Println(iam.ErrCodeMalformedPolicyDocumentException, aerr.Error())
			case iam.ErrCodeConcurrentModificationException:
				fmt.Println(iam.ErrCodeConcurrentModificationException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ""
	}

	fmt.Println("Role created: ",result)

	input2 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		RoleName:  aws.String(aws.StringValue(result.Role.RoleName)),
	}
	_, err = svc.AttachRolePolicy(input2)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ""
	}
	//fmt.Println("Attach cluster policy to Role: ", result2.String())

	input3 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSServicePolicy"),
		RoleName:  aws.String(aws.StringValue(result.Role.RoleName)),
	}
	_, err = svc.AttachRolePolicy(input3)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ""
	}
	//fmt.Println("Attach service policy to Role: ", result3.String())

	input4 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"),
		RoleName:  aws.String(aws.StringValue(result.Role.RoleName)),
	}
	_, err = svc.AttachRolePolicy(input4)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ""
	}
	//fmt.Println("Attach service policy to Role: ", result4.String())

	return	aws.StringValue(result.Role.Arn)


}
func get_role(sess *session.Session, clustername string) (string, string) {

	svc := iam.New(sess)
	input := &iam.GetRoleInput{
		RoleName:                 aws.String(clustername),
	}
	result, err := svc.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeEntityAlreadyExistsException:
				fmt.Println(iam.ErrCodeEntityAlreadyExistsException, aerr.Error())
			case iam.ErrCodeMalformedPolicyDocumentException:
				fmt.Println(iam.ErrCodeMalformedPolicyDocumentException, aerr.Error())
			case iam.ErrCodeConcurrentModificationException:
				fmt.Println(iam.ErrCodeConcurrentModificationException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", ""
	}

	fmt.Println("Role: ",result)
	return	aws.StringValue(result.Role.Arn), aws.StringValue(result.Role.RoleName)


}
func update_noderole(sess *session.Session, NodeRoleName string) {

	svc := iam.New(sess)

	input3 := &iam.UpdateAssumeRolePolicyInput{
		PolicyDocument: aws.String("{\n  \"Version\": \"2012-10-17\",\n  \"Statement\": [\n    {\n      \"Effect\": \"Allow\",\n      \"Principal\": {\n        \"Service\": \"eks.amazonaws.com\"\n      },\n      \"Action\": \"sts:AssumeRole\"\n    },\n    {\n      \"Effect\": \"Allow\",\n      \"Principal\": {\n        \"Service\": \"ec2.amazonaws.com\"\n      },\n      \"Action\": \"sts:AssumeRole\"\n    }\n  ]\n}"),
		RoleName:  &NodeRoleName,
	}
	_, err := svc.UpdateAssumeRolePolicy(input3)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	//fmt.Println("Attach EKSWorkerNodePolicy to Role: ", result3.String())

	input4 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"),
		RoleName:  &NodeRoleName,
	}
	_, err = svc.AttachRolePolicy(input4)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	//fmt.Println("Attach EKSWorkerNodePolicy to Role: ", result4.String())

	input5 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"),
		RoleName:  &NodeRoleName,
	}
	_, err = svc.AttachRolePolicy(input5)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	//fmt.Println("Attach EC2ContainerRegistryReadOnlyPolicy to Role: ", result5.String())

	input6 := &iam.AttachRolePolicyInput{
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"),
		RoleName:  &NodeRoleName,
	}
	_, err = svc.AttachRolePolicy(input6)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeLimitExceededException:
				fmt.Println(iam.ErrCodeLimitExceededException, aerr.Error())
			case iam.ErrCodeInvalidInputException:
				fmt.Println(iam.ErrCodeInvalidInputException, aerr.Error())
			case iam.ErrCodeUnmodifiableEntityException:
				fmt.Println(iam.ErrCodeUnmodifiableEntityException, aerr.Error())
			case iam.ErrCodePolicyNotAttachableException:
				fmt.Println(iam.ErrCodePolicyNotAttachableException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	//fmt.Println("Attach EKS_CNI_Policy to Role: ", result6.String())

	//return	aws.StringValue(result.Role.Arn)


}
func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	//os.Exit(1)
}
func ListStack_sdk(sess *session.Session, ClusterName string, NodegroupName string, Module string) (string, *eks.DescribeClusterOutput, *eks.DescribeNodegroupOutput, error, map[string]*string, []*eks.Taint) {

	var Status *string
	var result1 *eks.DescribeClusterOutput
	var result2 *eks.DescribeNodegroupOutput
	var err error
	time.Sleep(2 * time.Second)
	svc := eks.New(sess)
	if Module == "Master"{
		input := &eks.DescribeClusterInput{
			Name: aws.String(ClusterName),
		}
		result1, err := svc.DescribeCluster(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			Status = nil
			return "", nil, nil, err, nil, nil
		} else {
			Status = result1.Cluster.Status
			return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(awsutil.StringValue(Status), "\"",""),"\\","")), result1, nil, err, nil, nil

		}
	} else if Module == "NodeGroup"{
		input := &eks.DescribeNodegroupInput{
			ClusterName: aws.String(ClusterName),
			NodegroupName: aws.String(NodegroupName),
		}
		result2, err := svc.DescribeNodegroup(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeInvalidParameterException:
					fmt.Println(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					fmt.Println(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeServerException:
					fmt.Println(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeServiceUnavailableException:
					fmt.Println(eks.ErrCodeServiceUnavailableException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				fmt.Println(err.Error())
			}
			Status = nil
			return "", nil, nil, err, nil, nil
		} else {
			Status = result2.Nodegroup.Status
			return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(awsutil.StringValue(Status), "\"",""),"\\","")), nil, result2, err, nil, nil
		}
	}

		return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(awsutil.StringValue(Status), "\"",""),"\\","")),result1, result2, err, result2.Nodegroup.Labels, result2.Nodegroup.Taints
}
func readJSON(path string) (*map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: ", err)
	}
	contents := make(map[string]interface{})
	_ = json.Unmarshal(data, &contents)
	return &contents, nil
}
func difference(slice1 []*string, slice2 []*string) []string {
	var diff []string

	// Loop two times, first to find slice1 strings not in slice2,
	// second loop to find slice2 strings not in slice1
	for i := 0; i < 2; i++ {
		for _, s1 := range slice1 {
			found := false
			for _, s2 := range slice2 {
				if strings.TrimSpace(awsutil.StringValue(s1)) == strings.TrimSpace(awsutil.StringValue(s2)) {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(awsutil.StringValue(s1), "\\", ""), "\"", "")))
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			slice1, slice2 = slice2,slice1
		}
	}

	return diff
}


