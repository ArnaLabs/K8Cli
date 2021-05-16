package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/session"
	awscf "github.com/aws/aws-sdk-go/service/cloudformation"
	"os"
	"os/exec"
	"strings"
)

func checkHelmExists() {
	_, err := exec.LookPath("helm")
	if err != nil {
		fmt.Printf("didn't find 'helm' executable\n")
	}
	//else {
	//	fmt.Printf("'helm' executable is in '%s'\n", path)
	//}
}

func checkKubectlExists() {
	_, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Printf("didn't find 'kubectl' executable\n")
	}
	//else {
	//	fmt.Printf("'kubectl' executable is in '%s'\n", path)
	//}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
		//fmt.Printf("Listing stacks passed")
	}
	return resp
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	//os.Exit(1)
}

func ValidateStack(sess *session.Session, TemplateURL string, ElementsCreate map[string]string, ElementsUpdate map[string]string) ([]*awscf.Parameter, []*awscf.Parameter) {
	svc := awscf.New(sess)
	fmt.Printf("Validation session: ", svc)
	params := &awscf.ValidateTemplateInput{
		//TemplateBody: aws.String("TemplateBody"),
		TemplateURL: aws.String(TemplateURL),
	}
	resp, err := svc.ValidateTemplate(params)

	if err != nil {
		fmt.Println(os.Stderr, "Validation Failed with Error: %v\n", err)
		os.Exit(1)
	} else if err == nil {
		fmt.Printf("Stack validation passed")
	}
	fmt.Printf("Stack passed: ", awsutil.StringValue(resp))
	fmt.Printf("Number of Parameters defined in Stack: ", len(resp.Parameters))

	paramcreate := make([]*awscf.Parameter, len(resp.Parameters))
	paramupdate := make([]*awscf.Parameter, len(resp.Parameters))

	for i, p := range resp.Parameters {
		paramcreate[i] = &awscf.Parameter{
			ParameterKey: p.ParameterKey,
			//UsePreviousValue: aws.Bool(true),
			//Description: p.Description,
			//NoEcho:      p.NoEcho,
		}
		e := awsutil.StringValue(paramcreate[i].ParameterKey)
		k := strings.Trim(e, "\"")
		//Printf("test: ", elements[k])
		if ElementsCreate[k] != "" {
			paramcreate[i].ParameterValue = aws.String(ElementsCreate[k])
		} else {
			paramcreate[i].ParameterValue = p.DefaultValue
		}
	}

	for i, p := range resp.Parameters {
		paramupdate[i] = &awscf.Parameter{
			ParameterKey: p.ParameterKey,
			//UsePreviousValue: aws.Bool(true),
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
