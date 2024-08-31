# Amazon GuardDuty Event Response
Amazon GuardDuty is a service that can monitor threats to your AWS Cloud environments and is the focus of this example to showcase how go-event-bus-gen can be utilized.  See [AWS's Documentation](https://aws.amazon.com/guardduty/) for details on the service.

# Example
This example responds to any GuardDuty Finding and takes action on any instance finding and any access key finding.  In practice, this should not be used directly and should provide more conditions and logic to responding to events rather than blindly disabling access keys or stopping instances.  So disclaimer, please don't run this example in your own accounts without updating logic to fit your needs.  This example is intended as an example only of how to generate, run, and test an event bus using `go-event-bus-gen`

# Details
`main.go` is intended as a lambda function that responds to a cloudwatch event.  From that event, it is converted to a Finding.  If that Finding is an AccessKey finding, it will use the event bus to disable the access key.  If that finding has details on an instance, then it will stop that EC2 instance.

`handler.go` provides the implementation of the service that is used with the event bus

`guardduty.proto` provides the specification for the event bus and provided structs

`bus.go` is the generated logic utilized for running the event bus

`awsmocks.go` and `mocks.go` are generated mocks to use for testing

`main_test.go` is the test file utilized

`config.yaml` contains the imports for the aws events golang package for utilizing cloudwatch events in the generated `bus.go`