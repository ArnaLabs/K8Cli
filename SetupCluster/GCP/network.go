package GCP

import (
	"context"
	"fmt"
	"strings"

	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

func (g *GcpClient) ApplyNetwork(ctx context.Context) (*computepb.Network, error) {
	fmt.Println("Applying VPC Network")

	name := g.Cluster.Cloud.Cluster + "-vpc"

	vpc, err := g.GetVPC(ctx, name)
	if err != nil {
		return nil, err
	}

	if vpc == nil {
		fmt.Println("VPC Doesn't exist; creating")
		g.CreateVPC(ctx, name)
		vpc, err = g.GetVPC(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	// subnets are created automatically by gcp
	if *vpc.AutoCreateSubnetworks {
		return vpc, nil
	}

	expectedSubnets := []string{}
	for subnet := range g.Cluster.VPC.Subnets {
		expectedSubnets = append(expectedSubnets, subnet)
	}

	for _, subnet := range expectedSubnets {
		found := false
		// gcp only supports lowercase name for subnets
		subnetName := strings.ToLower(subnet)

		for _, existingSubnet := range vpc.Subnetworks {
			s := strings.Split(existingSubnet, "/")
			sn := s[len(s)-1]

			if sn == subnetName {
				found = true
				break
			}

		}

		if !found {
			err := g.CreateSubnet(ctx, vpc, subnet)
			if err != nil {
				return nil, err
			}
		}
	}

	return g.GetVPC(ctx, name)
}

func (g *GcpClient) GetVPC(ctx context.Context, name string) (*computepb.Network, error) {
	project := g.Cluster.Cloud.Project
	req := &computepb.GetNetworkRequest{
		Network: name,
		Project: project,
	}
	nw, err := g.NetworksClient.Get(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, err
	}
	return nw, nil
}

func (g *GcpClient) CreateVPC(ctx context.Context, name string) error {
	cluster := g.Cluster
	description := "VPC created by K8Cli for cluster " + cluster.Cloud.Cluster

	req := &computepb.InsertNetworkRequest{
		Project: cluster.Cloud.Project,
		NetworkResource: &computepb.Network{
			Name:                  &name,
			Description:           &description,
			AutoCreateSubnetworks: cluster.VPC.AutoCreateSubnetworks,
		},
	}

	op, err := g.NetworksClient.Insert(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (g *GcpClient) CreateSubnet(ctx context.Context, vpc *computepb.Network, name string) error {
	cluster := g.Cluster
	subnet := cluster.VPC.Subnets[name]
	fmt.Printf("Creating subnet %s %s\n", name, subnet)
	subnetName := strings.ToLower(name)

	req := &computepb.InsertSubnetworkRequest{
		Project: cluster.Cloud.Project,
		Region:  cluster.Cloud.Region,
		SubnetworkResource: &computepb.Subnetwork{
			Name:        &subnetName,
			IpCidrRange: &subnet,
			Network:     vpc.SelfLink,
			Region:      &cluster.Cloud.Region,
		},
	}
	op, err := g.SubnetworksClient.Insert(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}
	return nil
}

func (g *GcpClient) DeleteNetwork(ctx context.Context) error {
	fmt.Println("Deleting VPC")
	cluster := g.Cluster
	name := g.Cluster.Cloud.Cluster + "-vpc"

	vpc, err := g.GetVPC(ctx, name)
	if err != nil {
		return err
	}

	req := &computepb.DeleteNetworkRequest{
		Project: cluster.Cloud.Project,
		Network: vpc.GetName(),
	}

	op, err := g.NetworksClient.Delete(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}

	fmt.Println("VPC deleted")
	return nil
}
