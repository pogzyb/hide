package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"

	"github.com/pogzyb/hide/infra"
)

type AWSProvider struct {
	IpAddr         string
	VpcId          string
	PublicSubnetId string
}

func NewProvider(ctx context.Context, ipAddr, vpcId, subnetId string) (*AWSProvider, error) {
	if vpcId == "" {
		var err error
		vpcId, err = getDefaultVpc(ctx)
		if err != nil {
			return nil, err
		}
		log.Info().Msgf("using default vpc: %s", vpcId)
	}
	return &AWSProvider{
		IpAddr:         ipAddr,
		VpcId:          vpcId,
		PublicSubnetId: subnetId,
	}, nil
}

var (
	clientEC2 *ec2.Client
	userdata  = `#!/bin/bash
cd /home/ec2-user
wget https://github.com/pogzyb/hide/releases/download/0.1.0a/hide
chmod u+x ./hide
./hide serve --port 8181`
	defaultTags = []types.Tag{
		{Key: aws.String("CreatedBy"), Value: aws.String("hide-proxy")},
	}
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal().Msgf("could not get ec2 client: %v", err)
	}
	clientEC2 = ec2.NewFromConfig(cfg)
}

func (pr *AWSProvider) Deploy(ctx context.Context) (*infra.HideInstanceInfo, error) {
	securityGroupId, err := createSecurityGroup(ctx, pr.IpAddr, pr.VpcId)
	if err != nil {
		return nil, err
	}
	resp, err := createEC2(ctx, securityGroupId, &pr.PublicSubnetId)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second * 3)
	instanceId := *resp.Instances[0].InstanceId
	err = waitForEC2State(ctx, instanceId, 16)
	if err != nil {
		return nil, err
	}
	hostname, err := getEC2PublicDnsName(ctx, instanceId)
	if err != nil {
		return nil, err
	}
	info := &infra.HideInstanceInfo{
		Hostname: hostname,
		UID:      instanceId,
	}
	return info, nil
}

func (pr *AWSProvider) Destroy(ctx context.Context) error {
	if err := deleteEC2(ctx); err != nil {
		return err
	}
	time.Sleep(time.Second * 3)
	return deleteSecurityGroup(ctx)
}

func getDefaultVpc(ctx context.Context) (string, error) {
	resp, err := clientEC2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return "", err
	}
	for _, vpc := range resp.Vpcs {
		if *vpc.IsDefault {
			return *vpc.VpcId, nil
		}
	}
	return "", fmt.Errorf("could not find a default vpc")
}

func getAMI(ctx context.Context) (string, error) {
	resp, err := clientEC2.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"al2023-ami-2023*"},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{"arm64"},
			},
		},
		Owners: []string{"amazon"},
	})
	if err != nil {
		return "", nil
	}
	// todo: get most recent
	return *resp.Images[0].ImageId, nil
}

func createSecurityGroup(ctx context.Context, userIp, vpcId string) (string, error) {
	sgTags := []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("hide-proxy-sg")},
	}
	sgTags = append(sgTags, defaultTags[:]...)
	resp, err := clientEC2.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("hide-proxy-sg"),
		Description: aws.String("Controls traffic to hide-proxy."),
		VpcId:       aws.String(vpcId),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags:         sgTags,
			},
		},
	})
	if err != nil {
		return "", err
	}
	_, err = clientEC2.AuthorizeSecurityGroupEgress(
		ctx,
		&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId: resp.GroupId,
			IpPermissions: []types.IpPermission{
				{
					ToPort:     aws.Int32(80),
					FromPort:   aws.Int32(80),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp:      aws.String("0.0.0.0/0"),
							Description: aws.String("Allow HTTP internet access"),
						},
					},
				},
				{
					ToPort:     aws.Int32(443),
					FromPort:   aws.Int32(443),
					IpProtocol: aws.String("tcp"),
					IpRanges: []types.IpRange{
						{
							CidrIp:      aws.String("0.0.0.0/0"),
							Description: aws.String("Allow HTTPS internet access"),
						},
					},
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	_, err = clientEC2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: resp.GroupId,
		IpPermissions: []types.IpPermission{
			{
				ToPort:     aws.Int32(65535),
				FromPort:   aws.Int32(0),
				IpProtocol: aws.String("tcp"),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String(fmt.Sprintf("%s/32", userIp)),
						Description: aws.String("Allow access from your IP"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", err
	}
	return *resp.GroupId, nil
}

func findSecurityGroup(ctx context.Context) ([]types.SecurityGroup, error) {
	tagFilter := []types.Filter{
		{
			Name:   aws.String("tag:CreatedBy"),
			Values: []string{"hide-proxy"},
		},
	}
	resp, err := clientEC2.DescribeSecurityGroups(
		ctx,
		&ec2.DescribeSecurityGroupsInput{
			Filters: tagFilter,
		},
	)
	return resp.SecurityGroups, err
}

func deleteSecurityGroup(ctx context.Context) error {
	groups, err := findSecurityGroup(ctx)
	if err != nil {
		return err
	}
	for _, group := range groups {
		_, err = clientEC2.DeleteSecurityGroup(
			ctx,
			&ec2.DeleteSecurityGroupInput{GroupId: group.GroupId},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func createEC2(ctx context.Context, securityGroupId string, subnetId *string) (*ec2.RunInstancesOutput, error) {
	instanceTags := []types.Tag{
		{Key: aws.String("Name"), Value: aws.String("hide-proxy")},
	}
	instanceTags = append(instanceTags, defaultTags[:]...)
	imageId, err := getAMI(ctx)
	if err != nil {
		return nil, err
	}
	b64userdata := base64.StdEncoding.EncodeToString([]byte(userdata))
	return clientEC2.RunInstances(ctx, &ec2.RunInstancesInput{
		MaxCount:         aws.Int32(1),
		MinCount:         aws.Int32(1),
		ImageId:          aws.String(imageId),
		InstanceType:     types.InstanceTypeT4gNano,
		UserData:         aws.String(b64userdata),
		SecurityGroupIds: []string{securityGroupId},
		SubnetId:         subnetId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         instanceTags,
			},
		},
	})
}

func filterInstancesByTag(ctx context.Context) (*ec2.DescribeInstancesOutput, error) {
	return clientEC2.DescribeInstances(
		ctx,
		&ec2.DescribeInstancesInput{
			Filters: []types.Filter{
				{
					Name:   aws.String("tag:CreatedBy"),
					Values: []string{"hide-proxy"},
				},
			},
		},
	)
}

func findEC2(ctx context.Context) ([]string, error) {
	resp, err := filterInstancesByTag(ctx)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, res := range resp.Reservations {
		for _, instance := range res.Instances {
			ids = append(ids, *instance.InstanceId)
		}
	}
	return ids, nil
}

func deleteEC2(ctx context.Context) error {
	instanceIds, err := findEC2(ctx)
	if err != nil {
		return err
	}
	_, err = clientEC2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return err
	}
	for _, instanceId := range instanceIds {
		ctxInner, cancel := context.WithDeadline(ctx, time.Now().Add(time.Minute*5))
		err = waitForEC2State(ctxInner, instanceId, 48)
		cancel()
		if err != nil {
			return err
		}
	}
	return err
}

func getEC2PublicDnsName(ctx context.Context, instanceId string) (string, error) {
	resp, err := clientEC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return "", err
	}
	var dnsName string
	for _, res := range resp.Reservations {
		for _, inst := range res.Instances {
			dnsName = *inst.PublicDnsName
		}
	}
	return dnsName, nil
}

func waitForEC2State(ctx context.Context, instanceId string, stateCode int32) error {
	var currentCode int32
	for currentCode != stateCode {
		resp, err := clientEC2.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{
			InstanceIds:         []string{instanceId},
			IncludeAllInstances: aws.Bool(true),
		})
		if err != nil {
			return err
		}
		if len(resp.InstanceStatuses) == 0 {
			break
		}
		currentCode = *resp.InstanceStatuses[0].InstanceState.Code
		time.Sleep(time.Second * 5)
	}
	return nil
}
