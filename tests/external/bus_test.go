package external

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type EventBusTestSuite struct {
	suite.Suite
	service *MockService
}

func (suite *EventBusTestSuite) SetupTest() {
	suite.service = NewMockService(gomock.NewController(suite.T()))
}

func (suite *EventBusTestSuite) TestExample() {
	now := time.Now()
	gomock.InOrder(
		suite.service.EXPECT().HandleEvent(events.CloudWatchEvent{
			Time:       now,
			DetailType: "MyDetailType",
			Region:     "us-west-2",
		}).Return(nil),
	)

	bus := NewEventBus()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bus.Run(ctx, suite.service); err != nil {
			panic(err)
		}
	}()

	bus.Ready()

	err := bus.Publish(events.CloudWatchEvent{
		Time:       now,
		DetailType: "MyDetailType",
		Region:     "us-west-2",
	})
	assert.Nil(suite.T(), err)
	wg.Wait()
}

func TestEventBusTestSuite(t *testing.T) {
	suite.Run(t, new(EventBusTestSuite))
}
