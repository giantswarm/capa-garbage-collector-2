package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	capa "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AwsClient struct {
	client client.Client
}

func NewAwsClient(client client.Client) *AwsClient {
	return &AwsClient{
		client: client,
	}
}

func (awsClient *AwsClient) Get(ctx context.Context, namespacedName types.NamespacedName) (*capa.AWSCluster, error) {
	var awsCluster capa.AWSCluster
	if err := awsClient.client.Get(ctx, namespacedName, &awsCluster); err != nil {
		return nil, err
	}
	return &awsCluster, nil
}

func (awsClient *AwsClient) GetOwnerCluster(ctx context.Context, awsCluster *capa.AWSCluster) (*capi.Cluster, error) {
	cluster, err := util.GetOwnerCluster(ctx, awsClient.client, awsCluster.ObjectMeta)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}
