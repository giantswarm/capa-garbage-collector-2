package controllers_test

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ec2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	capa "sigs.k8s.io/cluster-api-provider-aws/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/giantswarm/capa-garbage-collector/controllers"
	"github.com/giantswarm/capa-garbage-collector/controllers/controllersfakes"
)

var _ = Describe("CapaGarbageCollectorReconciler", func() {
	var (
		ctx context.Context

		reconciler *controllers.CapaGarbageCollectorReconciler
		k8sClient  *controllersfakes.FakeAwsClusterClient
		awsClient  *controllersfakes.FakeAwsEc2Client

		cluster      *capi.Cluster
		awsCluster   *capa.AWSCluster
		result       ctrl.Result
		reconcileErr error
	)
	BeforeEach(func() {
		logger := zap.New(zap.WriteTo(GinkgoWriter))
		ctx = log.IntoContext(context.Background(), logger)

		k8sClient = new(controllersfakes.FakeAwsClusterClient)
		awsClient = new(controllersfakes.FakeAwsEc2Client)

		reconciler = &controllers.CapaGarbageCollectorReconciler{
			AwsClClient: k8sClient,
			AwsEc2Cl:    awsClient,
		}

		awsCluster = &capa.AWSCluster{}
		k8sClient.GetReturns(awsCluster, nil)

		cluster = &capi.Cluster{}
		k8sClient.GetOwnerClusterReturns(cluster, nil)
	})

	JustBeforeEach(func() {
		request := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "foo",
				Namespace: "bar",
			},
		}
		result, reconcileErr = reconciler.Reconcile(ctx, request)
	})

	It("gets aws cluster", func() {
		Expect(k8sClient.GetCallCount()).To(Equal(1))
		Expect(k8sClient.GetOwnerClusterCallCount()).To(Equal(1))

		_, actualCluster := k8sClient.GetOwnerClusterArgsForCall(0)
		Expect(actualCluster).To(Equal(awsCluster))
	})

	It("returns directly if not deleted", func() {
		Expect(result).To(Equal(ctrl.Result{}))
		Expect(reconcileErr).To(BeNil())
	})

	When("AwsCluster is marked for deletion", func() {
		BeforeEach(func() {
			awsCluster.Spec.NetworkSpec.VPC.ID = "vpc-test1234"
			now := metav1.Now()
			awsCluster.DeletionTimestamp = &now
		})

		It("uses aws client to get nginx security group", func() {
			Expect(awsClient.GetNginxControllerSecGroupCallCount()).To(Equal(1))
			vpcID := awsClient.GetNginxControllerSecGroupArgsForCall(0)
			Expect(vpcID).To(Equal("vpc-test1234"))
			Expect(result).To(Equal(ctrl.Result{}))
			Expect(reconcileErr).To(BeNil())
		})

		When("Nginx security group is found", func() {
			BeforeEach(func() {
				sgGroupName := "testsg"
				sg := &ec2.SecurityGroup{
					GroupName: &sgGroupName,
				}
				awsClient.GetNginxControllerSecGroupReturns(sg, nil)
			})

			It("uses aws client to delete security group", func() {
				Expect(awsClient.DeleteSecurityGroupCallCount()).To(Equal(1))
				Expect(result).To(Equal(ctrl.Result{}))
				Expect(reconcileErr).To(BeNil())
			})
		})
	})

})
