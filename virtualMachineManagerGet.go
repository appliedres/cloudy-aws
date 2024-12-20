package cloudyaws

import (
	"context"
	"fmt"

	"github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
)

func (vmm *AwsVirtualMachineManager) GetById(ctx context.Context, UVMID string) (*models.VirtualMachine, error) {
	log := logging.GetLogger(ctx)

	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:UVMID"),
				Values: []string{UVMID},
			},
		},
	}

	resp, err := vmm.vmClient.DescribeInstances(ctx, input)
	if err != nil {
		msg := fmt.Sprintf("GetByID Failed for UVMID: %s", UVMID)
		log.ErrorContext(ctx, msg)
		return nil, errors.Wrap(err, msg)
	}

	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		msg := fmt.Sprintf("GetByID VM not found: %s", UVMID)
		log.ErrorContext(ctx, msg)
		return nil, errors.Wrap(err, msg)
	}

	if len(resp.Reservations) != 1 && len(resp.Reservations[0].Instances) != 1 {
		msg := fmt.Sprintf("GetByID More than one VM found with UVMID: %s", UVMID)
		log.ErrorContext(ctx, msg)
		return nil, errors.Wrap(err, msg)
	}

	instance := resp.Reservations[0].Instances[0]
	vm := ToCloudyVirtualMachine(instance)

	return vm, nil
}

func (vmm *AwsVirtualMachineManager) GetAll(ctx context.Context, filter string, attrs []string) (*[]models.VirtualMachine, error) {
	vmList := []models.VirtualMachine{}

	input := &ec2.DescribeInstancesInput{}
	paginator := ec2.NewDescribeInstancesPaginator(vmm.vmClient, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return &vmList, err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				cloudyVm := ToCloudyVirtualMachine(instance)
				vmList = append(vmList, *cloudyVm)
			}
		}
	}

	return &vmList, nil
}
