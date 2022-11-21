/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ec2"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	capa "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

//counterfeiter:generate . AwsClusterClient
type AwsClusterClient interface {
	Get(context.Context, types.NamespacedName) (*capa.AWSCluster, error)
	GetOwnerCluster(context.Context, *capa.AWSCluster) (*capi.Cluster, error)
}

//counterfeiter:generate . AwsEc2Client
type AwsEc2Client interface {
	GetNginxControllerSecGroup(string) (*ec2.SecurityGroup, error)
	DeleteSecurityGroup(*ec2.SecurityGroup) error
}

// CapaGarbageCollectorReconciler reconciles a AwsCluster object
type CapaGarbageCollectorReconciler struct {
	AwsClClient AwsClusterClient
	AwsEc2Cl    AwsEc2Client
	Scheme      *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete

func (r *CapaGarbageCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling")
	defer logger.Info("Done reconciling")
	logger.Info("get cluster")
	awsCluster, err := r.AwsClClient.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("Aws Cluster does not exist anymore")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	logger.Info("get owner 2")
	cluster, err := r.AwsClClient.GetOwnerCluster(context.TODO(), awsCluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info("AWS Cluster does not have an owner cluster yet")
		return ctrl.Result{}, nil
	}

	if !awsCluster.DeletionTimestamp.IsZero() {
		logger.Info("Reconciling delete - cleanup")
		return r.reconcileDelete(ctx, awsCluster)
	}

	return ctrl.Result{}, nil
}

func (r *CapaGarbageCollectorReconciler) reconcileDelete(ctx context.Context, awsCluster *capa.AWSCluster) (ctrl.Result, error) {
	nginxSG, err := r.AwsEc2Cl.GetNginxControllerSecGroup(awsCluster.Spec.NetworkSpec.VPC.ID)
	if err != nil {
		return ctrl.Result{}, err
	} else if nginxSG == nil {
		return ctrl.Result{}, nil
	}

	if err := r.AwsEc2Cl.DeleteSecurityGroup(nginxSG); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CapaGarbageCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&capa.AWSCluster{}).
		Complete(r)
}
