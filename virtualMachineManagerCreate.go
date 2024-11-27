package cloudyaws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	"github.com/pkg/errors"
)

func (vmm *AwsVirtualMachineManager) Create(ctx context.Context, vm *models.VirtualMachine) (*models.VirtualMachine, error) {
	log := logging.GetLogger(ctx)

	log.InfoContext(ctx, "VM Create checking for existing VM with the same name")
	existingVM, err := vmm.FindVMByName(ctx, vm.Name)
	if err != nil {
		return nil, errors.Wrap(err, "VM Create: error checking for existing VM")
	}
	if existingVM != nil {
		return nil, fmt.Errorf("VM Create: a VM with the name '%s' already exists", vm.Name)
	}

	log.InfoContext(ctx, "VM Create starting")

	if vm.Location == nil {
		vm.Location = &models.VirtualMachineLocation{
			Cloud:  "aws",
			Region: vmm.credentials.Location,
		}
	}

	if vm.Tags == nil {
		vm.Tags = make(map[string]*string)
	}
	vm.Tags["CreatedBy"] = aws.String("cloudy-aws")
	vm.Tags["UVMID"] = aws.String(vm.ID)

	log.InfoContext(ctx, "VM Create setting up networking")
	nics, err := vmm.GetNics(ctx, vm.ID)
	if err != nil {
		return nil, errors.Wrap(err, "VM Create")
	} else if len(nics) != 0 {
		vm.Nics = nics
	} else {
		newNic, err := vmm.CreateNic(ctx, vm)
		if err != nil {
			return nil, errors.Wrap(err, "VM Create")
		}
		vm.Nics = append(vm.Nics, newNic)
	}

	log.InfoContext(ctx, "VM Create converting from cloudy to AWS")
	runInstancesInput := FromCloudyVirtualMachine(ctx, vm)

	log.InfoContext(ctx, "VM Create running instance")
	runOutput, err := vmm.vmClient.RunInstances(ctx, runInstancesInput)
	if err != nil {
		return nil, errors.Wrap(err, "VM Create: starting instance")
	}

	if len(runOutput.Instances) == 0 {
		return nil, fmt.Errorf("VM Create: no instance created")
	}

	instance := runOutput.Instances[0]

	log.InfoContext(ctx, fmt.Sprintf("VM Create instance created with ID: %s", *instance.InstanceId))

	log.InfoContext(ctx, "VM Create waiting for instance to reach 'running' state")
	waiter := ec2.NewInstanceRunningWaiter(vmm.vmClient)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{*instance.InstanceId},
	}, time.Minute*7)
	if err != nil {
		return nil, errors.Wrap(err, "VM Create: waiting for running state")
	}

	log.InfoContext(ctx, "VM Create instance running")

	updatedVM, err := UpdateCloudyVirtualMachine(vm, &instance)
	if err != nil {
		return nil, errors.Wrap(err, "VM Create")
	}

	return updatedVM, nil
}
