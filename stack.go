package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/session"
	awscf "github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/K8-Cloud/k8-cloud/cfts"
	"os"
	"strconv"
	"strings"
)

func ListStack(sess *session.Session, c cfts.Cftvpc, stackcreate []*awscf.Parameter, stackupdate []*awscf.Parameter) {
	type ByAge []cfts.Config
	var v = c
	var count = 0
	svc := awscf.New(sess)
	params := &awscf.DescribeStacksInput{}
	resp, err := svc.DescribeStacks(params)

	if err != nil {
		fmt.Println(os.Stderr, "Validation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Printf("Checking Stacks.......")
	}

	value := awsutil.StringValue(len(resp.Stacks))
	i, _ := strconv.Atoi(value)

	fmt.Printf("Number of Cloud Formation Templates exists:", i)

	if i == 0 {
		fmt.Printf("No stacks exist, creating stack")
		Createcft(sess, v, stackcreate)
		//j := 0
		fmt.Println("Checking Status.......")
		for {
			var a string = "CREATE_IN_PROGRESS"
			b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
			//print(b)
			var c string = strings.Trim(b, "\"")
			//var b string = strings.Trim(DescribeStack(sess, c.StackName, j), "\"")
			if a != c {
				fmt.Printf("Status: ", b)
				//time.Sleep(10 * time.Second)
				break
			}
		}
		fmt.Printf("Creation Completed")

	} else if i != 0 {
		present := make([]cfts.Config, int(i))
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
		fmt.Printf("Count :", count)
		if count == 1 {
			for k := 0; k < i; k++ {
				stacks, _ := strconv.Unquote(awsutil.StringValue(resp.Stacks[k].StackName))
				if stacks == c.StackName {
					//j := k
					fmt.Printf("Stack exist, updating stack")
					UpdateStack(sess, v, stackupdate)
					fmt.Printf("Checking Status.......")
					for {
						var a string = "UPDATE_IN_PROGRESS"
						b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
						var c string = strings.Trim(b, "\"")
						if a != c {
							fmt.Printf("Status: ", b)
							//time.Sleep(10 * time.Second)
							break
						}
					}
					fmt.Printf("Update Completed")
				}
			}
		} else if count == 0 {
			//j := 0
			fmt.Printf("Stack doesn't exist, creating stack")
			Createcft(sess, v, stackcreate)
			fmt.Printf("Checking Status.......")
			for {
				var a string = "CREATE_IN_PROGRESS"
				b := awsutil.StringValue(CheckStack(sess, c.StackName).Stacks[0].StackStatus)
				//print(b)
				var c string = strings.Trim(b, "\"")
				//var b string = strings.Trim(DescribeStack(sess, c.StackName, j), "\"")
				if a != c {
					fmt.Printf("Status: ", b)
					//time.Sleep(10 * time.Second)
					break
				}
			}
			fmt.Printf("Create Completed")
		}
	}
}

func Createcft(sess *session.Session, d cfts.Cftvpc, stack []*awscf.Parameter) *awscf.CreateStackOutput {
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
	fmt.Printf("Stack paramenters for creating stack :", params)
	rep, err := svc.CreateStack(params)

	if err != nil {
		fmt.Println(os.Stderr, "Creation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Printf("Stack Creation passed")
	}

	//fmt.Printf(awsutil.StringValue(rep))

	return rep
}

func UpdateStack(sess *session.Session, u cfts.Cftvpc, stack []*awscf.Parameter) *awscf.UpdateStackOutput {
	svc := awscf.New(sess)
	//fmt.Printf(stack)
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
	fmt.Printf("Stack paramenters for updating stack :", params)
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
