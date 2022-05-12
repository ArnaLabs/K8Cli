package GCP

import (
	"context"
	"fmt"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (g *GcpClient) UpdateCluster(ctx context.Context, cl *containerpb.Cluster) error {
	if cl.GetCurrentMasterVersion() != g.Cluster.Master.KubernetesVersion {
		fmt.Println("k8s master version changed, updating")
		err := g.UpdateMaster(ctx, cl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GcpClient) UpdateMaster(ctx context.Context, cl *containerpb.Cluster) error {
	req := &containerpb.UpdateClusterRequest{
		Name: cl.SelfLink,
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
