package gcp

import (
	"context"
	"fmt"
	"strings"

	compute "cloud.google.com/go/compute/apiv1"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"gopkg.in/yaml.v2"
)

func ApplyCluster(clusterYaml []byte) error {

  var cluster Cluster
  ctx := context.Background()

  if err := yaml.Unmarshal(clusterYaml, &cluster); err != nil {
    return err
  }

  if cluster.Cloud.Project == "" {
    return fmt.Errorf("Project id not defined")
  }

  fmt.Printf("applying gcp cluster with config: %v\n", cluster)

  var c *compute.NetworksClient
  var err error
  project := cluster.Cloud.Project

  if cluster.Cloud.CredentialsPath != "" {
    c, err = compute.NewNetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
    if err != nil {
      return err
    }
    defer c.Close()
  } else{
    fmt.Println("Using default credentials for google cloud")
    c, err = compute.NewNetworksRESTClient(ctx)
    if err != nil {
      return err
    }
    defer c.Close()
  }

  name:= cluster.Cloud.Cluster + "-vpc"

  exists, err := CheckVpcExists(ctx, c, name, project)
  if err != nil {
    return err
  }

  if exists{
    fmt.Println("VPC already exists")
  }else{
    description:= "VPC created by K8Cli for cluster "+cluster.Cloud.Cluster

    req := &computepb.InsertNetworkRequest{
      Project: cluster.Cloud.Project,
      NetworkResource: &computepb.Network{
        Name: &name,
        Description: &description,
        AutoCreateSubnetworks: cluster.VPC.AutoCreateSubnetworks,
      },
    }

    op, err := c.Insert(ctx, req)
    if err != nil {
      return err
    }

    if err := op.Wait(ctx);err != nil {
      return err
    }
  }

  return nil
}

func CheckVpcExists(ctx context.Context, c *compute.NetworksClient, name, project string) (bool, error){
  req := &computepb.GetNetworkRequest{
    Network: name,
    Project: project,
  }
  _, err := c.Get(ctx, req)
  if err != nil {
    if strings.Contains(err.Error(), "404") {
      return false, nil
    }
    return false, err
  }
  return true, nil
}
