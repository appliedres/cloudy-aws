package cloudyaws

import (
	"context"
	// "fmt"

	// "github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	// "github.com/pkg/errors"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func (vmm *AwsVirtualMachineManager) GetById(ctx context.Context, id string) (*models.VirtualMachine, error) {
	// 	log := logging.GetLogger(ctx)

	// 	resp, err := vmm.vmClient.Get(ctx, vmm.credentials.ResourceGroup, id, &armcompute.VirtualMachinesClientGetOptions{
	// 		Expand: nil,
	// 	})

	// 	if err != nil {
	// 		if is404(err) {
	// 			log.InfoContext(ctx, fmt.Sprintf("GetById vm not found: %s", id))
	// 			return nil, nil
	// 		}

	// 		return nil, errors.Wrap(err, "VM GetById")
	// 	}

	// 	vm := ToCloudyVirtualMachine(&resp.VirtualMachine)

	// 	return vm, nil

	return nil, nil
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