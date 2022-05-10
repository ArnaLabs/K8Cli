package GCP

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v2"

	compute "cloud.google.com/go/compute/apiv1"
	container "cloud.google.com/go/container/apiv1"
	"google.golang.org/api/option"
)

type GcpClient struct {
	NetworksClient       *compute.NetworksClient
	SubnetworksClient    *compute.SubnetworksClient
	ClusterManagerClient *container.ClusterManagerClient
	Cluster              *Cluster
}

func FromYaml(clusterYaml []byte) (error, *GcpClient) {
	var cluster Cluster
	ctx := context.Background()

	if err := yaml.Unmarshal(clusterYaml, &cluster); err != nil {
		return err, nil
	}

	if cluster.Cloud.Project == "" {
		return fmt.Errorf("Project id not defined"), nil
	}

	fmt.Printf("applying gcp cluster with config: %v\n", cluster)

	var nwClient *compute.NetworksClient
	var snClient *compute.SubnetworksClient
	var cmClient *container.ClusterManagerClient
	var err error

	if cluster.Cloud.CredentialsPath != "" {
		nwClient, err = compute.NewNetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err, nil
		}
		snClient, err = compute.NewSubnetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err, nil
		}
		cmClient, err = container.NewClusterManagerClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return err, nil
		}
	} else {
		fmt.Println("Using default credentials for google cloud")
		nwClient, err = compute.NewNetworksRESTClient(ctx)
		if err != nil {
			return err, nil
		}
		snClient, err = compute.NewSubnetworksRESTClient(ctx)
		if err != nil {
			return err, nil
		}
		cmClient, err = container.NewClusterManagerClient(ctx)
		if err != nil {
			return err, nil
		}
	}

	return nil, &GcpClient{
		NetworksClient:       nwClient,
		SubnetworksClient:    snClient,
		ClusterManagerClient: cmClient,
		Cluster:              &cluster,
	}
}

func (g *GcpClient) Apply() error {
	fmt.Println("Applying GKE")
	ctx := context.Background()

	if err := g.ApplyNetwork(ctx); err != nil {
		return err
	}

	return nil
}
