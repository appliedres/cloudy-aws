package cloudyaws

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/appliedres/cloudy"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)



// implements an exponential backoff with time.Sleep() to limit and spread calls out over time
func expBackoff(ctx context.Context, iteration int, max_ms int) {

	rand_ms, _ := rand.Int(rand.Reader, big.NewInt(1000))
	rand_ms_i := int(rand_ms.Uint64())

	sq := int(math.Exp2(float64(iteration-2)))
	base_ms := sq*500

	delay_ms := int(math.Min(float64(base_ms + rand_ms_i), float64(max_ms + rand_ms_i)))
	
	// cloudy.Info(ctx, "exponential backoff: %d ms, base_ms: %d, rand_ms: %d, n:%d, sq:%d", delay_ms, base_ms, rand_ms_i, iteration, sq)
	time.Sleep(time.Duration(delay_ms) * time.Millisecond)
}

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
	// fmt.Printf("ListALL start")

	var err error

	var returnList []*cloudyvm.VirtualMachineStatus

	all, err := vmc.Ec2Client.DescribeInstances(nil)

	if err != nil {
		fmt.Println("DescribeInstances error", err)
		return nil, err
	}

	for _, reservation := range all.Reservations {
		for _, instance := range reservation.Instances {
			// fmt.Printf("Found instance: %s\n", *instance.InstanceId)

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
			// fmt.Printf("Found instance: %s\n", *instance.InstanceId)

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
		err = cloudy.Error(ctx, "more than one instance found with ID '%s', returning only the first", instanceID)
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
		err = cloudy.Error(ctx, "invalid state requested: %s", state)
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
	if instStatus == nil {
		return cloudy.Error(ctx, "VM not found, could not stop")
	}
	if instStatus.PowerState == "running" {
		cloudy.Info(ctx, "instance already running")
		return nil
	}
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

	// err = waitForStatus(ctx, vmc, vmName, "running")
	// if err != nil {
	// 	return err
	// }

	return nil
}

func StopInstance(ctx context.Context, vmc *AwsEc2Controller, vmName string, wait bool) error {
	var err error
	var instStatus *cloudyvm.VirtualMachineStatus

	// TODO: VM name not found
	instStatus, err = vmc.Status(ctx, vmName)
	if instStatus == nil {
		return cloudy.Error(ctx, "VM not found, could not stop")
	}
	if instStatus.PowerState == "stopped" {
		cloudy.Info(ctx, "instance already stopped")
		return nil
	}
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

	// err = waitForStatus(ctx, vmc, vmName, "stopped")
	// if err != nil {
	// 	return err
	// }

	return nil
}

// creates an instance with a given vm config
func CreateInstance(ctx context.Context, vmc *AwsEc2Controller, vm *cloudyvm.VirtualMachineConfiguration) error {
	cloudy.Info(ctx, "[%s] creating instance", vm.ID)

	instanceOptions := &ec2.RunInstancesInput{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sdh"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize: aws.Int64(100),
				},
			},
		},
		ImageId:      aws.String("ami-0ab0629dba5ae551d"),     // TODO: make dynamic, this is hardcoded for Ubuntu Server 22.04
		InstanceType: aws.String(vm.SizeRequest.SpecificSize), // TODO: use Size.Name or SizeRequest.SpecificSize?
		// KeyName:      aws.String("my-key-pair"),
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
		// SecurityGroupIds: []*string{
		// 	aws.String("sg-1a2b3c4d"),
		// },
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Name),
					},
				},
			},
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:        aws.Int64(0), // Primary NIC
				NetworkInterfaceId: &vm.PrimaryNetwork.ID,
			},
		},
		KeyName: aws.String("manualTest"),
	}

	runResult, err := vmc.Ec2Client.RunInstances(instanceOptions)
	if err != nil {
		fmt.Println("Could not create instance: ", err)
		// TODO: delete created NIC, EBS, etc on create failure
		return err
	}
	// TODO: do we need to store instance ID?

	// err = waitForStatus(ctx, vmc, vm.Name, "running")
	// if err != nil {
	// 	return err
	// }

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)

	return nil
}

// Terminates an instance with a given name
func TerminateInstance(ctx context.Context, vmc *AwsEc2Controller, vmName string, wait bool) error {
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

	// err = waitForStatus(ctx, vmc, vmName, "terminated")
	// if err != nil {
	// 	return err
	// }

	return nil
}
