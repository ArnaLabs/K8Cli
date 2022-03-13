package gcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
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

	var nwClient *compute.NetworksClient
	var snClient *compute.SubnetworksClient
	var cmClient *container.ClusterManagerClient
	var err error
	project := cluster.Cloud.Project

	if cluster.Cloud.CredentialsPath != "" {
		nwClient, err = compute.NewNetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err
		}
		defer nwClient.Close()
		snClient, err = compute.NewSubnetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err
		}
		defer snClient.Close()
		cmClient, err = container.NewClusterManagerClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err
		}
		defer cmClient.Close()
	} else {
		fmt.Println("Using default credentials for google cloud")
		nwClient, err = compute.NewNetworksRESTClient(ctx)
		if err != nil {
			return err
		}
		defer nwClient.Close()
		snClient, err = compute.NewSubnetworksRESTClient(ctx)
		if err != nil {
			return err
		}
		defer snClient.Close()
		cmClient, err = container.NewClusterManagerClient(ctx)
		if err != nil {
			return err
		}
		defer cmClient.Close()
	}

	name := cluster.Cloud.Cluster + "-vpc"

	vpc, err := GetVPC(ctx, nwClient, name, project)
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

		op, err := nwClient.Insert(ctx, req)
		if err != nil {
			return err
		}

		if err := op.Wait(ctx); err != nil {
			return err
		}

		vpc, err = GetVPC(ctx, nwClient, name, project)
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

	if err := CreateCluster(ctx, cmClient, cluster, vpc); err != nil {
		return err
	}

	return nil
}

func CreateCluster(ctx context.Context, c *container.ClusterManagerClient, cluster Cluster, vpc *computepb.Network) error {

	name := "projects/" + cluster.Cloud.Project + "/locations/" + cluster.Cloud.Region + "/clusters/" + cluster.Cloud.Cluster
	// TODO use first param for updates
	_, err := c.GetCluster(ctx, &containerpb.GetClusterRequest{
		Name: name,
	})
	if err == nil {
		fmt.Println("cluster already exists, returning")
		return nil
	}
	if err != nil {
		// Some unknown error
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}

	network := *vpc.SelfLink
	subnetwork := vpc.Subnetworks[0]

	nodePools := []*containerpb.NodePool{}

	for _, node := range cluster.Nodes {

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

	parent := "projects/" + cluster.Cloud.Project + "/locations/" + cluster.Cloud.Region
	req := &containerpb.CreateClusterRequest{
		Parent: parent,
		Cluster: &containerpb.Cluster{
			Name:                  cluster.Cloud.Cluster,
			Description:           "Cluster " + cluster.Cloud.Cluster + " created by K8cli",
			InitialClusterVersion: cluster.Master.KubernetesVersion,
			Network:               network,
			Subnetwork:            subnetwork,
			NodePools:             nodePools,
			ResourceLabels:        cluster.Cluster.Labels,
			PrivateClusterConfig: &containerpb.PrivateClusterConfig{
				EnablePrivateNodes: cluster.Cluster.PrivateNodes,
			},
			Autoscaling: &containerpb.ClusterAutoscaling{},
			VerticalPodAutoscaling: &containerpb.VerticalPodAutoscaling{
				Enabled: cluster.Cluster.VPA,
			},
			LoggingConfig:    getLoggingConfig(cluster),
			MonitoringConfig: getMonitoringConfig(cluster),
		},
	}

	op, err := c.CreateCluster(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Cluster creation initiated, status : %d\nop:%v\n", op.GetStatus(), op)

	return WaitForOperation(ctx, c, cluster.Cloud.Project, op)

}

func getLoggingConfig(c Cluster) *containerpb.LoggingConfig {
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

func getMonitoringConfig(c Cluster) *containerpb.MonitoringConfig {
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

func WaitForOperation(ctx context.Context, c *container.ClusterManagerClient, project string, op *containerpb.Operation) error {
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, op.GetZone(), op.Name)

	fmt.Printf("Waiting for operation to complete : %s\n", name)

	for {
		o, err := c.GetOperation(ctx, &containerpb.GetOperationRequest{
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
