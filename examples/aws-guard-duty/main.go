package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

//go:generate go-event-bus-gen --in guardduty.proto --out bus.go --config config.yaml
//go:generate mockgen -source=bus.go -destination mocks.go -package main
//go:generate mockgen -source=handler.go -destination awsmocks.go -package main

var bus *EventBus

func HandleRequest(ctx context.Context, event *events.CloudWatchEvent) error {
	if event == nil {
		return fmt.Errorf("received nil event")
	}
	return bus.Publish(event)
}

func main() {
	bus := NewEventBus()

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(err)
	}

	svc := Handler{
		bus:       bus,
		iamClient: iam.NewFromConfig(cfg),
		ec2Client: ec2.NewFromConfig(cfg),
	}

	go bus.Run(context.Background(), &svc)
	bus.Ready()

	lambda.Start(HandleRequest)
}
