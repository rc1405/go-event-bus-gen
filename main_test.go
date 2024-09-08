package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type EventBusTestSuite struct {
	suite.Suite
}

func (suite *EventBusTestSuite) SetupTest() {
}

func TestEventBusTestSuite(t *testing.T) {
	suite.Run(t, new(EventBusTestSuite))
}

func (suite *EventBusTestSuite) TestAllTheTypes() {
	protof := `syntax = "proto3";
import "google/protobuf/empty.proto";
import "google/protobuf/any.proto";
import "google/protobuf/timestamp.proto";
package types;

message TypeRequest {
    double                    p1 = 0;
	float                     p2 = 1;
	int32                     p3 = 2;
	int64                     p4 = 3;
	uint32                    p5 = 4;
	uint64                    p6 = 5;
	sint32                    p7 = 6;
	sint64                    p8 = 7;
	fixed32                   p9 = 8;
	fixed64                   p10 = 9;
	sfixed32                  p11 = 10;
	sfixed64                  p12 = 11;
	optional bool             p13 = 12;
	repeated string           p14 = 13;
	bytes                     p15 = 14;
	google.protobuf.Any       p16 = 15;
	google.protobuf.Timestamp p17 = 16;
}

service TypeService {
  rpc HelloType (typeRequest) returns (google.protobuf.Empty) {}
  rpc HelloTime (typeRequest) returns (google.protobuf.Timestamp) {}
}`

	tmpl, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), tmpl, Template{
		Package: "types",
		Structs: []Struct{
			{
				Name: "TypeRequest",
				Attributes: []Attribute{
					{
						Name:    "P1",
						Type:    "float64",
						RawName: "p1",
					},
					{
						Name:    "P2",
						Type:    "float32",
						RawName: "p2",
					},
					{
						Name:    "P3",
						Type:    "int32",
						RawName: "p3",
					},
					{
						Name:    "P4",
						Type:    "int64",
						RawName: "p4",
					},
					{
						Name:    "P5",
						Type:    "uint32",
						RawName: "p5",
					},
					{
						Name:    "P6",
						Type:    "uint64",
						RawName: "p6",
					},
					{
						Name:    "P7",
						Type:    "int32",
						RawName: "p7",
					},
					{
						Name:    "P8",
						Type:    "int64",
						RawName: "p8",
					},
					{
						Name:    "P9",
						Type:    "uint32",
						RawName: "p9",
					},
					{
						Name:    "P10",
						Type:    "uint64",
						RawName: "p10",
					},
					{
						Name:    "P11",
						Type:    "int32",
						RawName: "p11",
					},
					{
						Name:    "P12",
						Type:    "int64",
						RawName: "p12",
					},
					{
						Name:     "P13",
						Type:     "bool",
						RawName:  "p13",
						Optional: true,
					},
					{
						Name:     "P14",
						Type:     "string",
						RawName:  "p14",
						Repeated: true,
					},
					{
						Name:    "P15",
						Type:    "[]byte",
						RawName: "p15",
					},
					{
						Name:    "P16",
						Type:    "any",
						RawName: "p16",
					},
					{
						Name:    "P17",
						Type:    "time.Time",
						RawName: "p17",
					},
				},
			},
		},
		Methods: []Method{
			{
				Name:      "HelloType",
				Input:     "TypeRequest",
				HasOutput: false,
			},
			{
				Name:      "HelloTime",
				Input:     "TypeRequest",
				HasOutput: true,
				Output:    "time.Time",
			},
		},
		Imports: []string{"time"},
	})
}

func (suite *EventBusTestSuite) TestMaps() {
	protof := `syntax = "proto3";
import "google/protobuf/empty.proto";
package types;

message TypeRequest {
    map<string, string> p1 = 0;
	
}

service TypeService {
  rpc HelloType (typeRequest) returns (google.protobuf.Empty) {}
}`

	tmpl, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), tmpl, Template{
		Package: "types",
		Structs: []Struct{
			{
				Name: "TypeRequest",
				Attributes: []Attribute{
					{
						Name:    "P1",
						Type:    "map[string]string",
						RawName: "p1",
					},
				},
			},
		},
		Methods: []Method{
			{
				Name:      "HelloType",
				Input:     "TypeRequest",
				HasOutput: false,
			},
		},
		Imports: []string{},
	})
}

func (suite *EventBusTestSuite) TestEnums() {
	protof := `syntax = "proto3";
import "google/protobuf/empty.proto";
package types;

enum Status {
  SUCCESS = 0;
  FAILURE = 1;
}

message TypeRequest {
	Status status = 0;
}

service TypeService {
  rpc HelloType (typeRequest) returns (google.protobuf.Empty) {}
}`

	tmpl, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), tmpl, Template{
		Package: "types",
		Structs: []Struct{
			{
				Name: "TypeRequest",
				Attributes: []Attribute{
					{
						Name:    "Status",
						Type:    "StatusEnum",
						RawName: "status",
					},
				},
			},
		},
		Methods: []Method{
			{
				Name:      "HelloType",
				Input:     "TypeRequest",
				HasOutput: false,
			},
		},
		Imports: []string{},
		Enums: []Enum{
			{
				Name: "Status",
				Members: []EnumMember{
					{
						Name:  "SUCCESS",
						Index: "0",
					},
					{
						Name:  "FAILURE",
						Index: "1",
					},
				},
			},
		},
	})
}

func (suite *EventBusTestSuite) TestConflictingMethodInputs() {
	protof := `syntax = "proto3";
package types;

message TypeRequestA {
	Status status = 0;
}

message TypeRequestB {
	Status status = 0;
}

service TypeServiceA {
  rpc HelloType (TypeRequestA) returns (google.protobuf.Empty) {}
}

service TypeServiceB {
  rpc HelloType (TypeRequestB) returns (google.protobuf.Empty) {}
}`

	_, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Equal(suite.T(), err, fmt.Errorf("Method HelloType has multiple inputs: TypeRequestB | TypeRequestA"))
}

func (suite *EventBusTestSuite) TestConflictingMethodOutputs() {
	protof := `syntax = "proto3";
package types;

message TypeRequest {
	Status status = 0;
}

message TypeResponseA {
	Status status = 0;
}

message TypeResponseB {
	Status status = 0;
}

service TypeServiceA {
  rpc HelloType (TypeRequest) returns (TypeResponseA) {}
}

service TypeServiceB {
  rpc HelloType (TypeRequest) returns (TypeResponseB) {}
}`

	_, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Equal(suite.T(), err, fmt.Errorf("Method HelloType has multiple outputs: TypeResponseB | TypeResponseA"))
}

func (suite *EventBusTestSuite) TestConflictingMethodReturnSignatures() {
	protof := `syntax = "proto3";
package types;

message TypeRequest {
	Status status = 0;
}

message TypeResponse {
	Status status = 0;
}

service TypeServiceA {
  rpc HelloType (TypeRequest) returns (TypeResponse) {}
}

service TypeServiceB {
  rpc HelloType (TypeRequest) returns (google.protobuf.Empty) {}
}`

	_, err := New([]string{}, bytes.NewReader([]byte(protof)))
	assert.Equal(suite.T(), err, fmt.Errorf("Method HelloType has multiple return signatures"))
}
