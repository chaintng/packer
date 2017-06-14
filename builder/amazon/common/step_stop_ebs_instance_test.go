import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/mitchellh/multistep"
)

type FakeEC2Conn struct {
	FakeOut *string
	FakeErr awserr.Error
}

func (c FakeEC2Conn) StopInstances(input *ec2.StopInstancesInput) (*ec2.StopInstancesOutput, error) {
	return c.FakeOut, c.FakeErr
}

func FakeErrorCodeInvalidInstanceID() string {
	return "InvalidInstanceID.NotFound"
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

	var mockEC2Conn FakeEC2Conn
	mockEC2Conn.FakeOut = "InvalidInstanceID.NotFound: The instance ID 'i-...' does not exist"
	mockEC2Conn.FakeErr.Code() = FakeErrorCodeInvalidInstanceID()

	// Set up state bag for test using generated state.
	state := new(multistep.BasicStateBag)
	state.Put("ec2", mockEC2Conn)
	state.Put("ui", ui)
	state.Put("instance", FakeInstance)



	var testSubject = &StepStopEBSBackedInstance{
		delete: func(string, <-chan struct{}) error { return fmt.Errorf("!! Unit Test FAIL !!") },
		say:    func(message string) {},
		error:  func(e error) {},
	}

	stateBag := DeleteTestStateBagStepDeleteResourceGroup()

	var result = testSubject.Run(stateBag)
	if result != multistep.ActionHalt {
		t.Fatalf("Expected the step to return 'ActionHalt', but got '%d'.", result)
	}

	if _, ok := stateBag.GetOk(constants.Error); ok == false {
		t.Fatalf("Expected the step to set stateBag['%s'], but it was not.", constants.Error)
	}

	// These are teh states grabbed by step_stop_ebs_instance
	// ec2conn := state.Get("ec2").(*ec2.EC2)
	// instance := state.Get("instance").(*ec2.Instance)
	// ui := state.Get("ui").(packer.Ui)

}
