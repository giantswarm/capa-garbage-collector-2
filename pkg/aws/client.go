package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/awserrors"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/filter"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SecurityGroupService struct {
	ec2Client *ec2.EC2
}

func NewSecurityGroupService() *SecurityGroupService {
	sess, _ := session.NewSession()
	ec2Client := ec2.New(sess)

	return &SecurityGroupService{
		ec2Client: ec2Client,
	}
}

func (s *SecurityGroupService) GetNginxControllerSecGroup(vpcId string) (*ec2.SecurityGroup, error) {
	describeInput := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			filter.EC2.VPC(vpcId),
		},
	}
	describeOutput, err := s.ec2Client.DescribeSecurityGroups(describeInput)
	if err != nil {
		return nil, err
	}

	var nginxSG *ec2.SecurityGroup
	for _, group := range describeOutput.SecurityGroups {
		if strings.Contains(*group.GroupName, "k8s") {
			nginxSG = group
		}

		if nginxSG == nil {
			log.Log.Info("No SecurityGroup found")
			return nil, nil
		}
	}
	return nginxSG, nil
}

func (s *SecurityGroupService) DeleteSecurityGroup(nginxSG *ec2.SecurityGroup) error {

	if len(nginxSG.IpPermissions) > 0 {
		revokeIngressInput := &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       aws.String(*nginxSG.GroupId),
			IpPermissions: nginxSG.IpPermissions,
		}
		if _, err := s.ec2Client.RevokeSecurityGroupIngress(revokeIngressInput); err != nil {
			return err
		}
	}
	deletionInput := &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(*nginxSG.GroupId),
	}

	if _, err := s.ec2Client.DeleteSecurityGroup(deletionInput); awserrors.IsIgnorableSecurityGroupError(err) != nil {
		return err
	}
	return nil
}
