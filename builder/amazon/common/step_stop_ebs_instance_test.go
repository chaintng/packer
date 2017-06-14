// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See the LICENSE file in builder/azure for license information.

package common

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

func TestStepStopEbsInstance(t *testing.T) {
	stateBag := createTestStateBagStepStopEbsInstance()
	var testSubject = &StepStopEBSBackedInstance{
		SpotPrice:           "dollarydollars",
		DisableStopInstance: false,
	}
	result := testSubject.Run(stateBag)

	if result != multistep.ActionContinue {
		t.Fatalf("Expected the step to return 'ActionContinue', but got '%d'", result)
	}

}

// Session is a mock session which is used to hit the mock server
var Session = func() *session.Session {
	// server is the mock server that simply writes a 200 status back to the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return session.Must(session.NewSession(&aws.Config{
		DisableSSL: aws.Bool(true),
		Endpoint:   aws.String(server.URL),
	}))
}()

// NewMockClient creates and initializes a client that will connect to the
// mock server
func NewMockClient(cfgs ...*aws.Config) *ec2.EC2 {
	c := Session.ClientConfig("Mock", cfgs...)

	svc := &ec2.EC2{
		Client: client.New(
			*c.Config,
			metadata.ClientInfo{
				ServiceName:   "Mock",
				SigningRegion: c.SigningRegion,
				Endpoint:      c.Endpoint,
				APIVersion:    "2015-12-08",
				JSONVersion:   "1.1",
				TargetPrefix:  "MockServer",
			},
			c.Handlers,
		),
	}

	return svc
}

func createCodePtr(x int64) *int64 {
	return &x
}

func createStrPtr(x string) *string {
	return &x
}

func createAwsConfig() *aws.Config {
	return &aws.Config{}
}

func createStateChangeResponse(current string, previous string) []*ec2.InstanceStateChange {
	response_values := map[string]int{
		"pending":       0,
		"running":       16,
		"shutting-down": 32,
		"terminated":    48,
		"stopping":      64,
		"stopped":       80,
	}

	var CState *ec2.InstanceState
	*CState = ec2.InstanceState{
		Code: createCodePtr(response_values[current]),
		Name: createStrPtr(current),
	}

	var PState *ec2.InstanceState
	*PState = ec2.InstanceState{
		Code: createCodePtr(response_values[previous]),
		Name: createStrPtr(previous),
	}

	var ISC *ec2.InstanceStateChange
	*ISC = ec2.InstanceStateChange{
		CurrentState:  CState,
		InstanceId:    createStrPtr("IAmAnInstanceID"),
		PreviousState: PState,
	}
	var StoppingInstances []*ec2.InstanceStateChange
	StoppingInstances = append(StoppingInstances, ISC)
	return StoppingInstances
}

// wrap up nice and warm in an XML burrito
type XMLResponse struct {
	StopInstancesResponse *XMLResult
}

type XMLResult struct {
	StopInstancesResult *ec2.StopInstancesOutput
}

func createTestStateBagStepStopEbsInstance() multistep.StateBag {
	// Make a faked UI, instance, and ec2 conection
	var out, err bytes.Buffer
	var ui packer.Ui = &packer.BasicUi{
		Writer:      &out,
		ErrorWriter: &err,
	}
	FakeInstance := &ec2.Instance{
		InstanceId: aws.String("instance-id"),
	}

	conf := createAwsConfig()
	ec2conn := NewMockClient(conf)
	ec2conn.Handlers.Clear()
	ec2conn.Handlers.Send.PushFront(func(r *request.Request) {
		var buf bytes.Buffer
		var dummyOutput = &ec2.StopInstancesOutput{
			StoppingInstances: createStateChangeResponse("stopping", "running"),
		}
		var dummyResponse = &XMLResponse{StopInstancesResponse: &XMLResult{StopInstancesResult: dummyOutput}}
		enc := xml.NewEncoder(&buf)
		enc.Encode(dummyResponse)

		r.HTTPResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewReader(buf.Bytes())),
			// this for UnmarshalMetaHandler
			Header: http.Header{"X-Amzn-Requestid": []string{"12345254232"}},
		}
	})

	state := new(multistep.BasicStateBag)
	state.Put("ec2", ec2conn)
	state.Put("ui", ui)
	state.Put("instance", FakeInstance)

	return state
}
