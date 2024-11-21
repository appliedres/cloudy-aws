package cloudyaws

import (
	"context"
	// "fmt"

	// "github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	// "github.com/pkg/errors"
	"github.com/aws/aws-sdk-go/service/ec2"
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
	err := vmm.vmClient.DescribeInstancesPagesWithContext(ctx, input,
		func(page *ec2.DescribeInstancesOutput, lastPage bool) bool {
			for _, reservation := range page.Reservations {
				for _, instance := range reservation.Instances {
					cloudyVm := ToCloudyVirtualMachine(instance)
					vmList = append(vmList, *cloudyVm)
				}
			}
			return !lastPage
		},
	)
	if err != nil {
		return &vmList, err
	}
	return &vmList, nil
}
