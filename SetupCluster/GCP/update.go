package GCP

import (
	"context"
	"fmt"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (g *GcpClient) UpdateCluster(ctx context.Context, cl *containerpb.Cluster) error {
	if !cl.AddonsConfig.HttpLoadBalancing.Disabled != g.Cluster.Cluster.HttpLoadBalancer {
		fmt.Println("k8s HTTP LB config changed, updating")
		err := g.UpdateAddons(ctx, cl)
		if err != nil {
			return err
		}
		return nil
	}

	if cl.GetCurrentMasterVersion() != g.Cluster.Master.KubernetesVersion {
		fmt.Println("k8s master version changed, updating")
		err := g.UpdateMaster(ctx, cl)
		if err != nil {
			return err
		}
	}

	fmt.Println("cluster up to date")

	return nil
}

func (g *GcpClient) UpdateAddons(ctx context.Context, cl *containerpb.Cluster) error {
	name := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region + "/clusters/" + cl.Name
	req := &containerpb.UpdateClusterRequest{
		Name: name,
		Update: &containerpb.ClusterUpdate{
			DesiredAddonsConfig: &containerpb.AddonsConfig{
				HttpLoadBalancing: &containerpb.HttpLoadBalancing{
					Disabled: !g.Cluster.Cluster.HttpLoadBalancer,
				},
			},
		},
	}

	op, err := g.ClusterManagerClient.UpdateCluster(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Addons updation initiated, status : %d\nop:%v\n", op.GetStatus(), op)
	return g.WaitForClusterOperation(ctx, op)
}

func (g *GcpClient) UpdateMaster(ctx context.Context, cl *containerpb.Cluster) error {
	name := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region + "/clusters/" + cl.Name
	req := &containerpb.UpdateClusterRequest{
		Name: name,
		Update: &containerpb.ClusterUpdate{
			DesiredMasterVersion: g.Cluster.Master.KubernetesVersion,
		},
	}

	op, err := g.ClusterManagerClient.UpdateCluster(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Master version updation initiated, status : %d\nop:%v\n", op.GetStatus(), op)
	return g.WaitForClusterOperation(ctx, op)
}
