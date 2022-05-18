package GCP

import (
	"context"
	"fmt"
	"strings"
	"time"

	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (g *GcpClient) ApplyCluster(ctx context.Context, vpc *computepb.Network) error {
	cl, err := g.GetCluster(ctx)
	if err != nil {
		return err
	}
	if cl == nil {
		fmt.Println("cluster doesn't exist, creating")
		err := g.CreateCluster(ctx, vpc)
		if err != nil {
			return err
		}
		cl, err = g.GetCluster(ctx)
		if err != nil {
			return err
		}
		return nil
	}

	fmt.Println("cluster already exists, checking for changes")

	return g.UpdateCluster(ctx, cl)
}

func (g *GcpClient) GetCluster(ctx context.Context) (*containerpb.Cluster, error) {
	name := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region + "/clusters/" + g.Cluster.Cloud.Cluster
	cl, err := g.ClusterManagerClient.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: name,
	})
	if err != nil && strings.Contains(err.Error(), "NotFound") {
		// when not found, just return a nil object instead of error
		return nil, nil
	}
	return cl, err
}

func (g *GcpClient) CreateCluster(ctx context.Context, vpc *computepb.Network) error {
	network := *vpc.SelfLink
	subnetId := g.Cluster.Cluster.SubnetId
	subnetwork := strings.ToLower(subnetId)

	nodePools := []*containerpb.NodePool{}

	for _, node := range g.Cluster.Nodes {

		taints := []*containerpb.NodeTaint{}
		for _, t := range node.Taints {
			taints = append(taints, &containerpb.NodeTaint{
				Key:    t.Key,
				Value:  t.Value,
				Effect: containerpb.NodeTaint_Effect(containerpb.NodeTaint_Effect_value[t.Effect]),
			})
		}

		autoScaling := &containerpb.NodePoolAutoscaling{
			Enabled:      node.ScalingConfig.MinSize > 0 && node.ScalingConfig.MaxSize > 0,
			MinNodeCount: node.ScalingConfig.MinSize,
			MaxNodeCount: node.ScalingConfig.MaxSize,
		}

		nodePools = append(nodePools, &containerpb.NodePool{
			Name: node.NodeGroupName,
			Config: &containerpb.NodeConfig{
				MachineType: node.MachineType,
				DiskSizeGb:  node.DiskSize,
				Tags:        node.Tags,
				Labels:      node.Labels,
				Taints:      taints,
				Preemptible: node.SpotInstance,
			},
			InitialNodeCount: node.ScalingConfig.DesiredSize,
			Autoscaling:      autoScaling,
		})
	}

	parent := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region
	req := &containerpb.CreateClusterRequest{
		Parent: parent,
		Cluster: &containerpb.Cluster{
			Name:                  g.Cluster.Cloud.Cluster,
			Description:           "Cluster " + g.Cluster.Cloud.Cluster + " created by K8cli",
			InitialClusterVersion: g.Cluster.Master.KubernetesVersion,
			Network:               network,
			Subnetwork:            subnetwork,
			NodePools:             nodePools,
			ResourceLabels:        g.Cluster.Cluster.Labels,
			PrivateClusterConfig:  privateClusterConfig(g.Cluster),
			IpAllocationPolicy: &containerpb.IPAllocationPolicy{
				UseIpAliases: g.Cluster.Cluster.VPCNativeRouting,
				UseRoutes:    !g.Cluster.Cluster.VPCNativeRouting,
			},
			Autoscaling: &containerpb.ClusterAutoscaling{},
			VerticalPodAutoscaling: &containerpb.VerticalPodAutoscaling{
				Enabled: g.Cluster.Cluster.VPA,
			},
			LoggingConfig:    getLoggingConfig(g.Cluster),
			MonitoringConfig: getMonitoringConfig(g.Cluster),
			AddonsConfig: &containerpb.AddonsConfig{
				HttpLoadBalancing: &containerpb.HttpLoadBalancing{
					Disabled: !g.Cluster.Cluster.HttpLoadBalancer,
				},
			},
			ShieldedNodes: &containerpb.ShieldedNodes{
				Enabled: g.Cluster.Cluster.ShieldedNodes,
			},
		},
	}

	if g.Cluster.Cluster.NetworkPolicy {
		req.Cluster.NetworkPolicy = &containerpb.NetworkPolicy{
			Enabled:  true,
			Provider: containerpb.NetworkPolicy_CALICO,
		}
	}

	if g.Cluster.Cluster.WorkloadIdentity {
		req.Cluster.WorkloadIdentityConfig = &containerpb.WorkloadIdentityConfig{
			WorkloadPool: g.Cluster.Cloud.Project + "svc.id.goog",
		}
	}

	op, err := g.ClusterManagerClient.CreateCluster(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Cluster creation initiated, status : %d\nop:%v\n", op.GetStatus(), op)

	return g.WaitForClusterOperation(ctx, op)
}

func privateClusterConfig(cluster *Cluster) *containerpb.PrivateClusterConfig {
	if cluster.Cluster.PrivateClusterConfig == nil {
		return nil
	}
	return &containerpb.PrivateClusterConfig{
		EnablePrivateNodes:  true,
		MasterIpv4CidrBlock: cluster.Cluster.PrivateClusterConfig.ControlPlaneCidr,
		MasterGlobalAccessConfig: &containerpb.PrivateClusterMasterGlobalAccessConfig{
			Enabled: cluster.Cluster.PrivateClusterConfig.ControlPlaneGlobalAccess,
		},
	}
}

func (g *GcpClient) WaitForClusterOperation(ctx context.Context, op *containerpb.Operation) error {
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", g.Cluster.Cloud.Project, op.GetZone(), op.Name)

	fmt.Printf("Waiting for operation to complete : %s\n", name)

	for {
		o, err := g.ClusterManagerClient.GetOperation(ctx, &containerpb.GetOperationRequest{
			Name: name,
		})
		if err != nil {
			return err
		}
		if o.GetStatus() == containerpb.Operation_DONE {
			fmt.Println("Operation complete!")
			return nil
		}

		fmt.Printf("\nOperation not complete yet, sleeping 10s\n")
		time.Sleep(10 * time.Second)
	}

}

func getLoggingConfig(c *Cluster) *containerpb.LoggingConfig {
	if !c.Cluster.CloudLogging {
		return nil
	}
	return &containerpb.LoggingConfig{
		ComponentConfig: &containerpb.LoggingComponentConfig{
			EnableComponents: []containerpb.LoggingComponentConfig_Component{
				containerpb.LoggingComponentConfig_SYSTEM_COMPONENTS,
				containerpb.LoggingComponentConfig_WORKLOADS,
			},
		},
	}
}

func getMonitoringConfig(c *Cluster) *containerpb.MonitoringConfig {
	if !c.Cluster.CloudMonitoring {
		return nil
	}
	return &containerpb.MonitoringConfig{
		ComponentConfig: &containerpb.MonitoringComponentConfig{
			EnableComponents: []containerpb.MonitoringComponentConfig_Component{
				containerpb.MonitoringComponentConfig_SYSTEM_COMPONENTS,
			},
		},
	}
}

func (g *GcpClient) DeleteCluster(ctx context.Context) error {
	cluster, err := g.GetCluster(ctx)
	name := "projects/" + g.Cluster.Cloud.Project + "/locations/" + g.Cluster.Cloud.Region + "/clusters/" + g.Cluster.Cloud.Cluster
	if err != nil {
		return err
	}
	if cluster == nil {
		fmt.Println("cluester doesn't exist, probably already deleted")
		return nil
	}

	req := &containerpb.DeleteClusterRequest{
		Name: name,
	}

	op, err := g.ClusterManagerClient.DeleteCluster(ctx, req)

	if err != nil {
		return err
	}

	fmt.Printf("Cluster deletion initiated, status : %d\nop:%v\n", op.GetStatus(), op)

	return g.WaitForClusterOperation(ctx, op)
}
