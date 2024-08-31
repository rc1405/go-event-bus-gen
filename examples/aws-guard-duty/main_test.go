package main

import (
	context "context"
	_ "embed"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:embed test_data/c_c_activity.json
var ccActivity []byte

//go:embed test_data/malicious_caller.json
var maliciousCaller []byte

type EventBusTestSuite struct {
	suite.Suite
	iamClient *MockIamClient
	ec2Client *MockEc2Client
	now       time.Time
}

func (suite *EventBusTestSuite) SetupTest() {
	suite.iamClient = NewMockIamClient(gomock.NewController(suite.T()))
	suite.ec2Client = NewMockEc2Client(gomock.NewController(suite.T()))
	suite.now = time.Now()
}

func (suite *EventBusTestSuite) TestIAM() {
	suite.iamClient.EXPECT().UpdateAccessKey(gomock.Any(), &iam.UpdateAccessKeyInput{
		AccessKeyId: aws.String("GeneratedFindingAccessKeyId"),
		Status:      types.StatusTypeInactive,
		UserName:    aws.String("GeneratedFindingUserName"),
	}, gomock.Any()).Return(&iam.UpdateAccessKeyOutput{}, nil)

	bus := NewEventBus()
	event := events.CloudWatchEvent{
		Time:       suite.now,
		DetailType: "GuardDuty Finding",
		Region:     "us-west-2",
		Detail:     maliciousCaller,
	}

	handler := Handler{
		bus:       bus,
		iamClient: suite.iamClient,
		ec2Client: suite.ec2Client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bus.Run(ctx, &handler); err != nil {
			panic(err)
		}
	}()

	bus.Ready()

	err := bus.Publish(event)
	assert.Nil(suite.T(), err)
	wg.Wait()
}

func (suite *EventBusTestSuite) TestInstance() {
	suite.ec2Client.EXPECT().StopInstances(gomock.Any(), &ec2.StopInstancesInput{
		InstanceIds: []string{"i-99999999"},
	}, gomock.Any()).Return(&ec2.StopInstancesOutput{}, nil)

	bus := NewEventBus()
	event := events.CloudWatchEvent{
		Time:       suite.now,
		DetailType: "GuardDuty Finding",
		Region:     "us-west-2",
		Detail:     ccActivity,
	}

	handler := Handler{
		bus:       bus,
		iamClient: suite.iamClient,
		ec2Client: suite.ec2Client,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bus.Run(ctx, &handler); err != nil {
			panic(err)
		}
	}()

	bus.Ready()

	err := bus.Publish(event)
	assert.Nil(suite.T(), err)
	wg.Wait()
}

func TestEventBusTestSuite(t *testing.T) {
	suite.Run(t, new(EventBusTestSuite))
}
