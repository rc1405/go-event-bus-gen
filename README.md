# go-event-bus-gen
go-event-bus-gen is a Go code generator to build out event driven processing frameworks to fulfill the purpose of handling the boilerplate needed to establish and interact with messages on an event bus in a simliar fashion that you would get with your http style code generators.  

go-event-bus-gen utilizes [ProtoBuf](https://developers.google.com/protocol-buffers/docs/reference/proto3-spec) specifications to build the underlying structs, publishing, and processing.

## Installation
```
GO11MODULE=on go get github.com/rc1405/go-event-bus-gen
```

## Example
A Protocol Buffer file such as [Simple](./tests/simple/simple.proto)
```
syntax = "proto3";
import "google/protobuf/empty.proto";
package simple;

message HelloRequest {
    string name = 0;
}

message HelloReply {
  string message = 0;
}

service HelloService {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
  rpc HelloWorld (HelloReply) returns (google.protobuf.Empty) {}
}
```

will generate the following Interface that needs to be satisfied

```
type Service interface {
	SayHello(HelloRequest) (HelloReply, error)
	HelloWorld(HelloReply) error
}
```

### Running
```
bus := NewEventBus()
go bus.Run(context.Context, Service)

// waits for all subscriptions to be created and ready for processing
bus.Ready()

if err := bus.Publish(HelloRequest{Name: "Cheddar"}); err != nil {
    panic(err)
}
```

## Code Generation
### Inline comments
Use inline comments such as `//go:generate go-event-bus-gen --in simple.proto --out bus.go` which would take in the protobuf from `simple.proto` and generate code to `bus.go`.  Or using a configuration file:  `//go:generate go-event-bus-gen --in external.proto --out bus.go --config config.yaml`

### Command Line
`go-event-bus-gen --in simple.proto --out bus.go` which would take in the protobuf from `simple.proto` and generate code to `bus.go`.  Or using a configuration file:  `go-event-bus-gen --in external.proto --out bus.go --config config.yaml`

## Advanced Use
### Foreign Inputs
Given the usecase where I would like to use structs not defined in the protobuf file, I would need to specify the needed imports through a config file.
```
imports:
  - github.com/aws/aws-lambda-go/events
```

In this case, the AWS Golang Events package will be added to imports and then can be used in the protobuf file, such as:
```
service HelloService {
  rpc HandleEvent (events.CloudWatchEvent) returns (google.protobuf.Empty) {}
}
```

### Enums
Generating enums follows the same pattern as rpc code generation. i.e.
```
enum Status {
  SUCCESS = 0;
  FAILURE = 1;
}

message HelloReply {
  Status status  = 1;
}
```

Results in code generation such as
```
type StatusEnum int32

const (
	SUCCESS StatusEnum = 0
	FAILURE StatusEnum = 1
)

type HelloReply struct {
	Status    StatusEnum        `json:"status"`
}
```

## Limitations
### Multiple Services
Services within the same proto file are bundled into the same go interface.  i.e.

```
service HelloService {
  rpc SayHello (HelloRequest) returns (HelloReply) {}
  rpc HelloWorld (HelloReply) returns (google.protobuf.Empty) {}
}

service ShadowService {
  rpc AnotherHello(HelloRequest) returns (google.protobuf.Empty) {}
}
```

bundles into the following interface
```
type Service interface {
	SayHello(HelloRequest) (HelloReply, error)
	HelloWorld(HelloReply) error
	AnotherHello(HelloRequest) error
}
```

Note: this also limits multiple services are unable to have a method with the same normalized name

### Imports
Support for external proto imports is extremely limited.  The ones supported are:
* `google.protobuf.Timestamp`: which will translate types to `time.Time`
* `google.protobuf.Any`: which will translate types to `any`
* `google.protobuf.Empty`: which modifies the return signature for the method from `(Item, error)` to `error`