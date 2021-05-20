package cfts

import (
	"encoding/json"
	_ "encoding/json"
	"fmt"
	//_ "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	//_ "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	//_ "github.com/Azure/go-autorest/autorest"
	//_ "github.com/Azure/go-autorest/autorest/to"
	"github.com/smallfish/simpleyaml"
	_ "gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

var filename = "cf-aws-fmt.yml"

type Config struct {
	Key   string
	Value string
}

type Cftvpc struct {
	StackName   string
	TemplateURL string
}

//AKS Vars
//var (
//	ctx        = context.Background()
//	clientData clientInfo
//	authorizer autorest.Authorizer
//)

//type clientInfo struct {
//	SubscriptionID string
//	VMPassword     string
//}

//Setup AKS or EKS Cluster

func setupCluster() {

	////Reading inputs from yaml

	//filename := "cf-aws-fmt.yml" EKS
	filename := "cf-fmt.yaml-azure" // AKS
	source, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	yaml, err := simpleyaml.NewYaml(source)
	if err != nil {
		panic(err)
	}

	////Creating Elements
	//Start EKS Cluster elements
	//ElementsSubnetIDs := make(map[string]string)
	//var Acceesskey, Secretkey, ServicePrinciple, ResourceGroup, Region string
	var Acceesskey, Secretkey, Region string
	//var nodelen int
	//var nodelist []interface{}
	//var sess session.Session
	//End EKS Cluster elements

	//passing values for setting up connections

	//Start EKS Cluster session values
	Cloud, err := yaml.Get("Cloud").Get("Name").String()
	if Cloud == "AWS" {
		Acceesskey, _ = yaml.Get("Cloud").Get("AccessKey").String()
		Secretkey, _ = yaml.Get("Cloud").Get("SecretAccKey").String()
		Region, _ = yaml.Get("Cloud").Get("Region").String()
	}
	//End EKS Cluster elements session values
	//if Cloud == "Azure" {
	//	ServicePrinciple, _ = yaml.Get("Cloud").Get("ServicePrinciple").String()
	//	ResourceGroup, _ = yaml.Get("Cloud").Get("ResourceGroup").String()
	//	Region, _ = yaml.Get("Cloud").Get("Region").String()
	//}
	//start AKS Cluster session values
	//Cloud, err := yaml.Get("Cloud").Get("Name").String()
	//Acceesskey, err := yaml.Get("Cloud").Get("AccessKey").String()
	//Secretkey, err := yaml.Get("Cloud").Get("SecretAccKey").String()
	//Region, err := yaml.Get("Cloud").Get("Region").String()

	//Print EKS Cluster elements
	if Cloud == "AWS" {
		fmt.Printf("Cloud: %#v\n", Cloud)
		fmt.Printf("AccessKey: %#v\n", Acceesskey)
		fmt.Printf("SecAccKey: %#v\n", Secretkey)
		fmt.Printf("Region: %#v\n", Region)
		fmt.Printf("Creating sessions")
	}
	//Print AKS Cluster elements
	//if Cloud == "Azure" {
	//	fmt.Printf("Cloud: %#v\n", Cloud)
	//	fmt.Printf("ServicePrinciple: %#v\n", ServicePrinciple)
	//	fmt.Printf("ResourceGroup: %#v\n", ResourceGroup)
	//	fmt.Printf("Region: %#v\n", Region)
	//	fmt.Printf("Creating sessions")
	//}

	////Create Sessions
	//Create session EKS Cluster elements
	if Cloud == "AWS" {
		//sess, err := session.NewSession(&aws.Config{
		//	//aws.Config{throttle.Throttle()}
		//	Region:      aws.String(Region),
		//	Credentials: credentials.NewStaticCredentials(Acceesskey, Secretkey, ""),
		//})
	}
	//End session EKS Cluster elements
	//if Cloud == "Azure" {
	//	var err error
	//	authorizer, err = auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	//	if err != nil {
	//		log.Fatalf("Failed to get OAuth config: %v", err)
	//	}
	//
	//	authInfo, err := readJSON(os.Getenv("service-principle.auth"))
	//	if err != nil {
	//		log.Fatalf("Failed to read JSON: %+v", err)
	//	}
	//	clientData.SubscriptionID = (*authInfo)["subscriptionId"].(string)
	//	clientData.VMPassword = (*authInfo)["clientSecret"].(string)
	//}

	fmt.Printf("Session created ")

	////Checking if VPC is enabled

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
