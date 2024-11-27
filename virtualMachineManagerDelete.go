package cloudyaws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"

	"github.com/appliedres/cloudy/logging"
)

func (vmm *AwsVirtualMachineManager) Delete(ctx context.Context, vmID string) error {
	log := logging.GetLogger(ctx)

	if vmID == "" {
		return fmt.Errorf("cannot delete a VM without an ID")
	}

	log.InfoContext(ctx, "VM Delete locating instance with UVMID tag")

	// the vmID is stored as a UVMID tag on the instance
	describeInput := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:UVMID"),
				Values: []string{vmID},
			},
		},
	}

	output, err := vmm.vmClient.DescribeInstances(ctx, describeInput)
	if err != nil {
		return fmt.Errorf("VM Delete: error describing instances: %w", err)
	}

	// TODO: The first instance should always match the UVMID tag
	var instanceID string
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			instanceID = *instance.InstanceId
			log.InfoContext(ctx, fmt.Sprintf("VM Delete found instance with ID: %s, state: %s", instanceID, instance.State.Name))
			break
		}
	}
	if instanceID == "" {
		return fmt.Errorf("VM Delete: no instance found with UVMID tag value '%s'", vmID)
	}

	describeInstanceInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	instanceOutput, err := vmm.vmClient.DescribeInstances(ctx, describeInstanceInput)
	if err != nil {
		return fmt.Errorf("VM Delete: error checking instance state: %w", err)
	}

	for _, reservation := range instanceOutput.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name == types.InstanceStateNameTerminated {
				log.InfoContext(ctx, fmt.Sprintf("VM Delete instance with ID: %s is already terminated", instanceID))
				return nil
			}
		}
	}

	// Terminate the instance
	log.InfoContext(ctx, "VM Delete terminating instance")
	_, err = vmm.vmClient.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("VM Delete: error terminating instance: %w", err)
	}

	// wait for termination
	log.InfoContext(ctx, "VM Delete waiting for instance to terminate")
	waiter := ec2.NewInstanceTerminatedWaiter(vmm.vmClient)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("VM Delete: error waiting for termination: %w", err)
	}

	log.InfoContext(ctx, fmt.Sprintf("VM Delete successfully terminated instance with ID: %s", instanceID))

	log.InfoContext(ctx, "Starting GetNics")
	nics, err := vmm.GetNics(ctx, vmID)
	if err != nil {
		return errors.Wrap(err, "VM Delete")
	}

	if len(nics) > 0 {
		log.InfoContext(ctx, "Found NICs for VM, Starting DeleteNics", "count", len(nics), "id", vmID)
		err = vmm.DeleteNics(ctx, nics)
		if err != nil {
			return errors.Wrap(err, "NIC Delete")
		}
	} else {
		log.InfoContext(ctx, "No Nics found")
	}

	return nil
}
