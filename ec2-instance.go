package cloudyaws

import (
	"context"
	"fmt"
	"strings"

	"github.com/appliedres/cloudy"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// TODO: move vm logic to vm.go, limit this to instance logic only

func ValidateConfiguration(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) error {
	// TODO: This is common in AWS/Azure - move to cloudy?
	if strings.Contains(strings.ToLower(vm.OSType), "linux") {
	} else if strings.EqualFold(vm.OSType, "windows") {
	} else {
		return cloudy.Error(ctx, "[%s] invalid OS Type: %v, cannot create vm", vm.ID, vm.OSType)
	}

	return nil
}

func ListAllInstances(ctx context.Context, vmc *AwsEc2Controller) ([]*cloudyvm.VirtualMachineStatus, error) {
	fmt.Printf("ListALL start")

	var err error

	var returnList []*cloudyvm.VirtualMachineStatus

	all, err := vmc.Ec2Client.DescribeInstances(nil)

	if err != nil {
		fmt.Println("DescribeInstances error", err)
		return nil, err
	} else {
		fmt.Println("DescribeInstances success")
	}

	for _, reservation := range all.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Printf("Found instance: %s\n", *instance.InstanceId)

			vmStatus := &cloudyvm.VirtualMachineStatus{}

			// TODO: Handle no tags, instance could not have name tag or any tags
			vmStatus.Name = ""
			for _, t := range instance.Tags {
				if *t.Key == "Name" {
					vmStatus.Name = *t.Value
				}
			}
			vmStatus.PowerState = *instance.State.Name
			vmStatus.ID = *instance.InstanceId

			returnList = append(returnList, vmStatus)
		}
	}

	return returnList, err
}

func ListInstancesWithTag(ctx context.Context, vmc *AwsEc2Controller, tag string) ([]*cloudyvm.VirtualMachineStatus, error) {
	// TODO: list with tags, see ListAll
	return nil, nil
}

func InstanceStatusByID(ctx context.Context, vmc *AwsEc2Controller, instanceID string) (*cloudyvm.VirtualMachineStatus, error) {
	cloudy.Info(ctx, "retrieving status for instance ID '%s'", instanceID)
	var err error

	var returnList []*cloudyvm.VirtualMachineStatus
	var result *cloudyvm.VirtualMachineStatus

	instances, err := vmc.Ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// TODO: case for instance ID not found
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	}

	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			fmt.Printf("Found instance: %s\n", *instance.InstanceId)

			vmStatus := &cloudyvm.VirtualMachineStatus{}

			vmStatus.Name = ""
			for _, t := range instance.Tags {
				if *t.Key == "Name" {
					vmStatus.Name = *t.Value
				}
			}
			vmStatus.PowerState = *instance.State.Name
			vmStatus.ID = *instance.InstanceId

			returnList = append(returnList, vmStatus)
		}
	}

	if len(returnList) > 1 {
		result = returnList[0]
		err = fmt.Errorf("more than one instance found with ID '%s', returning only the first", instanceID)
	} else if len(returnList) == 1 {
		result = returnList[0]
	} else {
		return nil, err
	}
	return result, err
}

// given a VM Name, find the status of the underlying instance
// The instance will have a name tag matching the VM Name
// returns nil if no matching instance found
func InstanceStatusByVmName(ctx context.Context, vmc *AwsEc2Controller, vmName string) (*cloudyvm.VirtualMachineStatus, error) {
	// VM ID is stored as Instance Name
	
	var err error

	var returnList []*cloudyvm.VirtualMachineStatus
	var result *cloudyvm.VirtualMachineStatus

	instances, err := vmc.Ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(vmName),
				},
			},
		},
	})
	
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return nil, err
	} else if instances == nil {
		cloudy.Info(ctx, "No instances found with Instance Name Tag '%s'", vmName)
		return nil, err
	}

	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			vmStatus := &cloudyvm.VirtualMachineStatus{}

			vmStatus.Name = ""
			for _, t := range instance.Tags {
				if *t.Key == "Name" {
					vmStatus.Name = *t.Value
				}
			}
			vmStatus.PowerState = *instance.State.Name
			vmStatus.ID = *instance.InstanceId

			cloudy.Info(ctx, "found instance '%s', status='%s'", vmStatus.ID, vmStatus.PowerState)
			returnList = append(returnList, vmStatus)
		}
	}

	if len(returnList) > 1 {
		result = returnList[0]
		err = fmt.Errorf("more than one instance found with name '%s', returning only the first", vmName)
	} else if len(returnList) == 1 {
		result = returnList[0]
	} else {
		return nil, err
	}
	return result, err
}

func SetInstanceState(ctx context.Context, vmc *AwsEc2Controller, state cloudyvm.VirtualMachineAction, vmName string, wait bool) (*cloudyvm.VirtualMachineStatus, error) {
	if ctx == nil {
		ctx = cloudy.StartContext()
	}

	var vmStatus *cloudyvm.VirtualMachineStatus
	var err error

	if state == cloudyvm.VirtualMachineStart {
		err = vmc.Start(ctx, vmName, wait)
	} else if state == cloudyvm.VirtualMachineStop {
		err = vmc.Stop(ctx, vmName, wait)
	} else if state == cloudyvm.VirtualMachineTerminate {
		err = vmc.Terminate(ctx, vmName, wait)
	} else {
		err = fmt.Errorf("invalid state requested: %s", state)
		return vmStatus, err
	}

	if err != nil {
		return nil, err
	}

	vmStatus, err = vmc.Status(ctx, vmName)

	return vmStatus, err
}

func StartInstance(ctx context.Context, vmc *AwsEc2Controller, vmName string, wait bool) error {
	var err error
	var instStatus *cloudyvm.VirtualMachineStatus

	instStatus, err = vmc.Status(ctx, vmName)

	if err != nil {
		return err
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(instStatus.ID),
		},
	}

	_, err = vmc.Ec2Client.StartInstances(input)

	if err != nil {
		return err
	}

	// TODO: wait for or verify instance started

	return nil
}

func StopInstance(ctx context.Context, vmc *AwsEc2Controller, vmName string, wait bool) error {
	var err error
	var instStatus *cloudyvm.VirtualMachineStatus

	instStatus, err = vmc.Status(ctx, vmName)

	if err != nil {
		return err
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(instStatus.ID),
		},
	}

	_, err = vmc.Ec2Client.StopInstances(input)

	if err != nil {
		return err
	}

	// TODO: wait for or verify instance stopped

	return nil
}

func TerminateVmInstance(ctx context.Context, vmc *AwsEc2Controller, vmName string, wait bool) error {
	// TODO: switch to using instance ID as it is properly unique
	cloudy.Info(ctx, "Terminating Instance with name '%s'", vmName)  
	var err error
	var vmStatus *cloudyvm.VirtualMachineStatus

	vmStatus, err = vmc.Status(ctx, vmName)
	if err != nil {
		return err
	} else if vmStatus == nil {
		// instance not found by name
		cloudy.Info(ctx, "Could not terminate instance, Instance named '%s' not found", vmName)
		return nil
	}

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(vmStatus.ID),
		},
	}

	_, err = vmc.Ec2Client.TerminateInstances(input)
	if err != nil {
		return err
	}

	// TODO: wait for or verify instance terminated

	return nil
}

func CreateInstance(ctx context.Context, vmc *AwsEc2Controller, vm *cloudyvm.VirtualMachineConfiguration) error {
	return nil
}

func DeleteInstance(ctx context.Context, vmc *AwsEc2Controller, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineConfiguration, error) {
	return nil, nil
}