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

func FromYaml(clusterYaml []byte) (*GcpClient, error) {
	var cluster Cluster
	ctx := context.Background()

	if err := yaml.Unmarshal(clusterYaml, &cluster); err != nil {
		return nil, err
	}

	if cluster.Cloud.Project == "" {
		return nil, fmt.Errorf("Project id not defined")
	}

	fmt.Printf("applying gcp cluster with config: %v\n", cluster)

	var nwClient *compute.NetworksClient
	var snClient *compute.SubnetworksClient
	var cmClient *container.ClusterManagerClient
	var err error

	if cluster.Cloud.CredentialsPath != "" {
		nwClient, err = compute.NewNetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return nil, err
		}
		snClient, err = compute.NewSubnetworksRESTClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return nil, err
		}
		cmClient, err = container.NewClusterManagerClient(ctx, option.WithServiceAccountFile(cluster.Cloud.CredentialsPath))
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("Using default credentials for google cloud")
		nwClient, err = compute.NewNetworksRESTClient(ctx)
		if err != nil {
			return nil, err
		}
		snClient, err = compute.NewSubnetworksRESTClient(ctx)
		if err != nil {
			return nil, err
		}
		cmClient, err = container.NewClusterManagerClient(ctx)
		if err != nil {
			return nil, err
		}
	}

	return &GcpClient{
		NetworksClient:       nwClient,
		SubnetworksClient:    snClient,
		ClusterManagerClient: cmClient,
		Cluster:              &cluster,
	}, nil
}

func (g *GcpClient) Apply() error {
	fmt.Println("Applying GKE")
	ctx := context.Background()

	if err := g.ApplyNetwork(ctx); err != nil {
		return err
	}

	return nil
}
