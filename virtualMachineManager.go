package cloudyaws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	// "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	// "github.com/aws/aws-sdk-go/service/pricing"
	// "github.com/aws/aws-sdk-go/service/sts"

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

	vmClient *ec2.EC2
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
	credential, err := NewAwsCredentials(vmm.credentials)
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(vmm.credentials.Region),
		Credentials: credential,
	})
	if err != nil {
		return err
	}

	// Initialize Aws service clients
	vmm.vmClient = ec2.New(sess)

	return nil
}

func (vmm *AwsVirtualMachineManager) Start(ctx context.Context, vmName string) error {
	log := logging.GetLogger(ctx)

	var err error
	var instStatus *cloudyvm.VirtualMachineStatus

	instStatus, err = vmm.Status(ctx, vmName)
	if instStatus == nil {
		return errors.Wrap(err, "VM not found, could not stop")
	}
	if instStatus.PowerState == "running" {
		log.InfoContext(ctx, "instance already running")
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "Error when checking VM status")
	}

	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(instStatus.ID),
		},
	}

	_, err = vmm.vmClient.StartInstances(input)
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
	// log := logging.GetLogger(ctx)

	// poller, err := vmm.vmClient.BeginPowerOff(ctx, vmm.credentials.ResourceGroup, vmName, &armcompute.VirtualMachinesClientBeginPowerOffOptions{})
	// if err != nil {
	// 	return errors.Wrap(err, "VM Stop")
	// }

	// _, err = pollWrapper(ctx, poller, "VM Stop")
	// if err != nil {
	// 	return errors.Wrap(err, "VM Stop")
	// }

	// log.InfoContext(ctx, "VM Stop complete")

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

// given a VM Name, find the status of the underlying instance
// The instance will have a name tag matching the VM Name
// returns nil if no matching instance found
func (vmm *AwsVirtualMachineManager) Status(ctx context.Context, vmName string) (*cloudyvm.VirtualMachineStatus, error) {
	// VM ID is stored as Instance Name
	log := logging.GetLogger(ctx)

	var err error

	var returnList []*cloudyvm.VirtualMachineStatus
	var result *cloudyvm.VirtualMachineStatus

	instances, err := vmm.vmClient.DescribeInstances(&ec2.DescribeInstancesInput{
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
		log.InfoContext(ctx, "No instances found with Instance Name Tag '%s'", vmName)
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

			returnList = append(returnList, vmStatus)
		}
	}

	if len(returnList) > 1 {
		result = returnList[0]
		err = errors.Wrap(err, fmt.Sprintf("more than one instance found with name '%s', returning only the first", vmName))
	} else if len(returnList) == 1 {
		result = returnList[0]
	} else {
		return nil, err
	}

	// TODO: nil status

	return result, err
}

// wait for a given VM to reach a specific status
func (vmm *AwsVirtualMachineManager) waitForStatus(ctx context.Context, vmName string, desired_status string) error {
	log := logging.GetLogger(ctx)

	// TODO: should timeout be added?
	timeStart := time.Now()
	n := 1
	for {
		status, err := vmm.Status(ctx, vmName)
		if err != nil {
			return err
		}

		if status.PowerState == desired_status {
			log.InfoContext(ctx, "[%s] VM status reached '%s' in %.2f seconds", vmName, desired_status, float64(time.Since(timeStart)/time.Millisecond)/1000.0)
			break
		}

		log.InfoContext(ctx, "[%s] waiting for instances to transition from '%s' to '%s'", vmName, status.PowerState, desired_status)
		expBackoff(ctx, n, 32000)
		n = n + 1
	}

	return nil
}
