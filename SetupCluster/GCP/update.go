package GCP

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func UpdateCluster(ctx context.Context, c *container.ClusterManagerClient, name string, cluster Cluster) error {

	req := &containerpb.UpdateClusterRequest{
		Name: name,
		Update: &containerpb.ClusterUpdate{
			DesiredMasterVersion: cluster.Master.KubernetesVersion,
			// DesiredVerticalPodAutoscaling: &containerpb.VerticalPodAutoscaling{
				// Enabled: cluster.Cluster.VPA,
			// },
			// DesiredLoggingConfig:    getLoggingConfig(cluster),
			// DesiredMonitoringConfig: getMonitoringConfig(cluster),
		},
	}

	op, err := c.UpdateCluster(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Cluster updation initiated, status : %d\nop:%v\n", op.GetStatus(), op)
	return WaitForOperation(ctx, c, cluster.Cloud.Project, op)
}
