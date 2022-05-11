package GCP

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func UpdateCluster(ctx context.Context, c *container.ClusterManagerClient, cl *containerpb.Cluster, cluster Cluster) error {
	name := "projects/" + cluster.Cloud.Project + "/locations/" + cluster.Cloud.Region + "/clusters/" + cluster.Cloud.Cluster

	fmt.Printf("got existing cluster, %+v\n", cl)

	masterVersion := cl.CurrentMasterVersion

	fmt.Printf("Master version : %s\n", masterVersion)

	if masterVersion != cluster.Master.KubernetesVersion {
		req := &containerpb.UpdateClusterRequest{
			Name: name,
			Update: &containerpb.ClusterUpdate{
				DesiredMasterVersion: cluster.Master.KubernetesVersion,
			},
		}

		op, err := c.UpdateCluster(ctx, req)
		if err != nil {
			return err
		}

		fmt.Printf("Master version updation initiated, status : %d\nop:%v\n", op.GetStatus(), op)
		// err = WaitForOperation(ctx, c, cluster.Cloud.Project, op)
		// if err != nil {
		// return err
		// }
	} else {
		fmt.Printf("Nothing to update")
	}

	return nil

}

func (g *GcpClient) UpdateMaster(ctx context.Context) error {
	name := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region + "/clusters/" + g.Cluster.Cloud.Cluster
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
