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
	var snClient *compute.SubnetworksClient
	var err error
	project := cluster.Cloud.Project

	if cluster.Cloud.CredentialsPath != "" {
		c, err = compute.NewNetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err
		}
		defer c.Close()
		snClient, err = compute.NewSubnetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err
		}
		defer snClient.Close()
	} else {
		fmt.Println("Using default credentials for google cloud")
		c, err = compute.NewNetworksRESTClient(ctx)
		if err != nil {
			return err
		}
		defer c.Close()
		snClient, err = compute.NewSubnetworksRESTClient(ctx)
		if err != nil {
			return err
		}
		defer snClient.Close()
	}

	name := cluster.Cloud.Cluster + "-vpc"

	vpc, err := GetVPC(ctx, c, name, project)
	if err != nil {
		return err
	}

	if vpc == nil {
		description := "VPC created by K8Cli for cluster " + cluster.Cloud.Cluster

		req := &computepb.InsertNetworkRequest{
			Project: cluster.Cloud.Project,
			NetworkResource: &computepb.Network{
				Name:                  &name,
				Description:           &description,
				AutoCreateSubnetworks: cluster.VPC.AutoCreateSubnetworks,
			},
		}

		op, err := c.Insert(ctx, req)
		if err != nil {
			return err
		}

		if err := op.Wait(ctx); err != nil {
			return err
		}

		vpc, err = GetVPC(ctx, c, name, project)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Got VPC : %s\n", *vpc.SelfLink)

	if !*cluster.VPC.AutoCreateSubnetworks {
		vpcUrl := vpc.SelfLink
		if err := CreateSubnets(ctx, snClient, cluster, vpcUrl); err != nil {
			return err
		}
	}

	return nil
}

func CreateSubnets(ctx context.Context, c *compute.SubnetworksClient, cluster Cluster, vpcUrl *string) error {
	subnets := cluster.VPC.Subnets
	fmt.Printf("Creating Subnets %v\n", subnets)

	for name, subnet := range subnets {
		name = strings.ToLower(name)
		req := &computepb.InsertSubnetworkRequest{
			Project: cluster.Cloud.Project,
			Region:  cluster.Cloud.Region,
			SubnetworkResource: &computepb.Subnetwork{
				Name:        &name,
				IpCidrRange: &subnet,
				Network:     vpcUrl,
				Region:      &cluster.Cloud.Region,
			},
		}
		op, err := c.Insert(ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "409") {
				// subnet is already there
				continue
			}
			return err
		}

		if err := op.Wait(ctx); err != nil {
			return err
		}
	}

	fmt.Println("subnets created")

	return nil
}

func GetVPC(ctx context.Context, c *compute.NetworksClient, name, project string) (*computepb.Network, error) {
	req := &computepb.GetNetworkRequest{
		Network: name,
		Project: project,
	}
	nw, err := c.Get(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return nw, nil
}
