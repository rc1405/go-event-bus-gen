package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type IamClient interface {
	UpdateAccessKey(context.Context, *iam.UpdateAccessKeyInput, ...func(*iam.Options)) (*iam.UpdateAccessKeyOutput, error)
}

type Ec2Client interface {
	StopInstances(ctx context.Context, params *ec2.StopInstancesInput, optFns ...func(*ec2.Options)) (*ec2.StopInstancesOutput, error)
}

type Handler struct {
	bus       *EventBus
	iamClient IamClient
	ec2Client Ec2Client
}

func (h *Handler) ParseEvent(event events.CloudWatchEvent) (Finding, error) {
	var finding Finding
	if err := json.Unmarshal(event.Detail, &finding); err != nil {
		return Finding{}, err
	}

	return finding, nil
}

func (h *Handler) Evaluate(finding Finding) error {
	switch {
	case finding.Resource.AccessKeyDetails.UserType == "IAMUser" && finding.Resource.ResourceType == "AccessKey":
		if err := h.bus.Publish(finding.Resource.AccessKeyDetails); err != nil {
			return err
		}
	case finding.Resource.ResourceType == "Instance":
		if err := h.bus.Publish(InstanceDetails{
			Region:     finding.Region,
			InstanceId: finding.Resource.InstanceDetails.InstanceId,
		}); err != nil {
			return err
		}
	default:
	}
	return nil
}

func (h *Handler) DisableAccessKey(key AccessKeyDetails) error {
	_, err := h.iamClient.UpdateAccessKey(context.Background(), &iam.UpdateAccessKeyInput{
		AccessKeyId: &key.AccessKeyId,
		Status:      types.StatusTypeInactive,
		UserName:    &key.UserName,
	}, func(opts *iam.Options) {
		opts.Region = "us-east-1"
	})

	return err
}

func (h *Handler) StopInstance(details InstanceDetails) error {
	_, err := h.ec2Client.StopInstances(context.Background(), &ec2.StopInstancesInput{
		InstanceIds: []string{details.InstanceId},
	}, func(opt *ec2.Options) {
		opt.Region = details.Region
	})

	return err
}
