package GCP

import (
	"context"
	"fmt"
	"strings"

	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
)

func (g *GcpClient) ApplyNetwork(ctx context.Context) error {
	fmt.Println("Applying VPC Network")

	name := g.Cluster.Cloud.Cluster + "-vpc"

	vpc, err := g.GetVPC(ctx, name)
	if err != nil {
		return err
	}

	if vpc == nil {
		fmt.Println("VPC Doesn't exist; creating")
		g.CreateVPC(ctx, name)
		vpc, err = g.GetVPC(ctx, name)
		if err != nil {
			return err
		}
	}

	// subnets are created automatically by gcp
	if *vpc.AutoCreateSubnetworks {
		return nil
	}

	return nil
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
