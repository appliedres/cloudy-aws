package cloudyaws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	cloudyvm "github.com/appliedres/cloudy/vm"
	"github.com/pkg/errors"
)

const (
	MIN_WINDOWS_OS_DISK_SIZE = 200
)

type AwsVirtualMachineManager struct {
	credentials *AwsCredentials
	config      *VirtualMachineManagerConfig

	vmClient *ec2.Client
	// nicClient    *armnetwork.InterfacesClient
	// diskClient   *armcompute.DisksClient
	// subnetClient *armnetwork.SubnetsClient

	// dataClient  *armcompute.ResourceSKUsClient
	// usageClient *armcompute.UsageClient

	// galleryClient *armcompute.SharedGalleryImageVersionsClient

	LogBody bool
}

func NewAwsVirtualMachineManager(ctx context.Context, credentials *AwsCredentials, config *VirtualMachineManagerConfig) (cloudyvm.VirtualMachineManager, error) {

	vmm := &AwsVirtualMachineManager{
		credentials: credentials,
		config:      config,

		LogBody: false,
	}
	err := vmm.Configure(ctx)
	if err != nil {
		return nil, err
	}

	return vmm, nil
}

func (vmm *AwsVirtualMachineManager) Configure(ctx context.Context) error {
	credProvider, err := NewAwsCredentials(vmm.credentials)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(ctx,
										 config.WithRegion(vmm.credentials.Region),
										 config.WithCredentialsProvider(credProvider),
										)
	if err != nil {
		return err
	}

	vmm.vmClient = ec2.NewFromConfig(cfg)

	return nil
}

func (vmm *AwsVirtualMachineManager) Start(ctx context.Context, vmName string) error {
	log := logging.GetLogger(ctx)

	var err error

	vm, err := vmm.GetByName(ctx, vmName)
	if err != nil {
		return errors.Wrap(err, "Error when checking VM status")
	}
	if vm.State == "" {
		return errors.Wrap(err, "VM status not found, could not stop")
	}
	if vm.State == "running" {
		log.InfoContext(ctx, "instance already running")
		return nil
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{vm.ID},
	}

	_, err = vmm.vmClient.StartInstances(ctx, input)
	if err != nil {
		return err
	}

	err = vmm.waitForStatus(ctx, vmName, "running")
	if err != nil {
		return err
	}

	return nil
}

func (vmm *AwsVirtualMachineManager) Stop(ctx context.Context, vmName string) error {
	log := logging.GetLogger(ctx)

	var err error

	vm, err := vmm.GetByName(ctx, vmName)
	if err != nil {
		return errors.Wrap(err, "Error when checking VM status")
	}

	if vm.State == "" {
		return errors.Wrap(err, "VM status not found, could not stop")
	}

	if vm.State == "stopped" {
		log.InfoContext(ctx, "Instance already stopped")
		return nil
	}

	input := &ec2.StopInstancesInput{
		InstanceIds: []string{vm.ID},
	}

	_, err = vmm.vmClient.StopInstances(ctx, input)
	if err != nil {
		return errors.Wrap(err, "Error stopping VM")
	}

	err = vmm.waitForStatus(ctx, vmName, "stopped")
	if err != nil {
		return err
	}

	return nil
}

func (vmm *AwsVirtualMachineManager) Deallocate(ctx context.Context, vmName string) error {
	// log := logging.GetLogger(ctx)

	// poller, err := vmm.vmClient.BeginDeallocate(ctx, vmm.credentials.ResourceGroup, vmName, &armcompute.VirtualMachinesClientBeginDeallocateOptions{})
	// if err != nil {
	// 	if is404(err) {
	// 		log.InfoContext(ctx, "BeginDeallocate - VM not found")
	// 		return nil
	// 	}

	// 	return errors.Wrap(err, "VM Deallocate")
	// }

	// _, err = pollWrapper(ctx, poller, "VM Deallocate")
	// if err != nil {
	// 	return errors.Wrap(err, "VM Deallocate")
	// }

	// log.InfoContext(ctx, "VM Deallocate complete")

	return nil
}

func (vmm *AwsVirtualMachineManager) Update(ctx context.Context, vm *models.VirtualMachine) (*models.VirtualMachine, error) {
	return nil, nil
}

// func UpdateCloudyVirtualMachine(vm *models.VirtualMachine, responseVirtualMachine armcompute.VirtualMachine) error {

// 	return nil
// }


// wait for a given VM to reach a specific status
func (vmm *AwsVirtualMachineManager) waitForStatus(ctx context.Context, vmName string, desired_status string) error {
	log := logging.GetLogger(ctx)

	// TODO: should timeout be added?
	timeStart := time.Now()
	n := 1
	for {
		vm, err := vmm.GetByName(ctx, vmName)
		if err != nil {
			return err
		}

		if vm.State == desired_status {
			log.InfoContext(ctx, "[%s] VM status reached '%s' in %.2f seconds", vmName, desired_status, float64(time.Since(timeStart)/time.Millisecond)/1000.0)
			break
		}

		log.InfoContext(ctx, "[%s] waiting for instances to transition from '%s' to '%s'", vmName, vm.State, desired_status)
		expBackoff(ctx, n, 32000)
		n = n + 1
	}

	return nil
}
