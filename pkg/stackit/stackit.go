package stackit

import (
	"github.com/aws/aws-sdk-go/service/cloudformation"
				"log"
	"fmt"
			"encoding/json"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/aws/awserr"
		)

type Stackit struct {
	api cloudformationiface.CloudFormationAPI
	stackName string
	stackId string
}

func NewStackit(api cloudformationiface.CloudFormationAPI, stackName string) *Stackit {
	return &Stackit{api: api, stackName: stackName}
}

func (s *Stackit) describe() (*cloudformation.Stack, error) {
	stackName := s.stackId
	if len(stackName) == 0 {
		stackName = s.stackName
	}

	resp, err := s.api.DescribeStacks(&cloudformation.DescribeStacksInput{StackName: &stackName})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			code := awsErr.Code()
			if code == "ThrottlingException" {
				return s.describe()
			} else if code == "ValidationError" {
				return nil, nil
			}
		}
		return nil, errors.Wrap(err, "determining stack status")
	}

	stack := resp.Stacks[0]
	s.stackId = *stack.StackId
	return stack, nil
}

func (s *Stackit) error(err error, events chan<- TailStackEvent) {
	events <- TailStackEvent{StackitError: err}
	close(events)
}

func (s *Stackit) PrintOutputs() {
	stack, err := s.describe()

	if err != nil {
		log.Fatal(err.Error())
	}

	outputMap := make(map[string]string)

	for _, output := range stack.Outputs {
		outputMap[*output.OutputKey] = *output.OutputValue
	}

	bytes, err := json.MarshalIndent(outputMap, "", "  ")
	fmt.Println(string(bytes))
}

func (s *Stackit) IsSuccessfulState() (bool, error) {
	stack, err := s.describe()
	if err != nil {
		return false, errors.Wrap(err, "determining stack status")
	}

	status := *stack.StackStatus
	return status != "CREATE_COMPLETE" && status != "UPDATE_COMPLETE", nil
}