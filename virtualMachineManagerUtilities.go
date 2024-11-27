package cloudyaws

import (
	"context"
	"github.com/pkg/errors"
	"fmt"
	"strings"

	"github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func FromCloudyVirtualMachine(ctx context.Context, vm *models.VirtualMachine) *ec2.RunInstancesInput {
	log := logging.GetLogger(ctx)

	tags := []types.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(vm.Name),
		},
	}
	if vm.Tags != nil {
		for k, v := range vm.Tags {
			tags = append(tags, types.Tag{
				Key:   aws.String(k),
				Value: v,
			})
		}
	}

	if vm.Template != nil && vm.Template.Tags != nil {
		for k, v := range vm.Template.Tags {
			exists := false
			for _, tag := range tags {
				if *tag.Key == k {
					exists = true
					break
				}
			}
			if !exists {
				tags = append(tags, types.Tag{
					Key:   aws.String(k),
					Value: v,
				})
			}
		}
	}

	runInstancesInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(vm.OsBaseImageID),
		InstanceType: types.InstanceType(vm.Template.Size.ID),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         tags,
			},
		},
		MinCount: aws.Int32(1),
		MaxCount: aws.Int32(1),
	}

	// TODO: configure os settings
	// switch vm.Template.OperatingSystem {
	// case "windows":
	// 	runInstancesInput.UserData = aws.String(cloudy.GenerateWindowsUserData(vm.Template.LocalAdministratorID))
	// case "linux":
	// 	runInstancesInput.UserData = aws.String(cloudy.GenerateLinuxUserData(vm.Template.LocalAdministratorID))
	// }

	// attach network interfaces
	if vm.Nics != nil {
		networkInterfaces := []types.InstanceNetworkInterfaceSpecification{}
		for _, nic := range vm.Nics {
			networkInterfaces = append(networkInterfaces, types.InstanceNetworkInterfaceSpecification{
				NetworkInterfaceId: aws.String(nic.ID),
				DeviceIndex:        aws.Int32(0),
			})
		}
		runInstancesInput.NetworkInterfaces = networkInterfaces
	}

	log.InfoContext(ctx, fmt.Sprintf("prepared EC2 instance input: %+v", runInstancesInput))
	return runInstancesInput
}

func ToCloudyVirtualMachine(instance types.Instance) *models.VirtualMachine {
	cloudyVm := models.VirtualMachine{
		ID:   aws.ToString(instance.InstanceId),
		Name: aws.ToString(instance.InstanceId), // AWS instances typically don't have a "name"; use tags if needed
		Location: &models.VirtualMachineLocation{
			Region: aws.ToString(instance.Placement.AvailabilityZone),
		},
		Template: &models.VirtualMachineTemplate{},
		Tags:     map[string]*string{},
	}

	// State
	if instance.State != nil {
		cloudyVm.State = string(instance.State.Name)
	}

	// Size
	cloudyVm.Template.Size = &models.VirtualMachineSize{
		Name: string(instance.InstanceType),
	}

	// Network Interfaces
	nics := []*models.VirtualMachineNic{}
	for _, nic := range instance.NetworkInterfaces {
		if nic.NetworkInterfaceId != nil {
			nics = append(nics, &models.VirtualMachineNic{ID: aws.ToString(nic.NetworkInterfaceId)})
		}
	}
	cloudyVm.Nics = nics

	// OS Disk (AWS doesn't explicitly differentiate OS Disk from others; using root volume)
	if instance.RootDeviceName != nil {
		for _, bd := range instance.BlockDeviceMappings {
			if aws.ToString(bd.DeviceName) == aws.ToString(instance.RootDeviceName) {
				if bd.Ebs != nil {
					cloudyVm.OsDisk = &models.VirtualMachineDisk{
						ID:     aws.ToString(bd.Ebs.VolumeId),
						OsDisk: true,
					}
				}
			}
		}
	}

	// Data Disks
	disks := []*models.VirtualMachineDisk{}
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			isOsDisk := cloudyVm.OsDisk != nil && cloudyVm.OsDisk.ID == aws.ToString(bd.Ebs.VolumeId)
			if !isOsDisk {
				disks = append(disks, &models.VirtualMachineDisk{
					ID:     aws.ToString(bd.Ebs.VolumeId),
					OsDisk: false,
				})
			}
		}
	}
	cloudyVm.Disks = disks

	// Tags
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			if strings.EqualFold(aws.ToString(tag.Key), "Name") {
				cloudyVm.Name = aws.ToString(tag.Value)
			} else {
				cloudyVm.Tags[aws.ToString(tag.Key)] = tag.Value
			}
		}
	}

	return &cloudyVm
}

// func ToCloudyVirtualMachineSize(ctx context.Context, resource *armcompute.ResourceSKU) *models.VirtualMachineSize {

// 	log := logging.GetLogger(ctx)

// 	size := models.VirtualMachineSize{
// 		ID:   *resource.Name,
// 		Name: *resource.Name,
// 		Family: &models.VirtualMachineFamily{
// 			ID:   *resource.Family,
// 			Name: *resource.Family,
// 		},
// 	}

// 	locations := map[string]*models.VirtualMachineLocation{}

// 	for _, location := range resource.Locations {
// 		_, ok := locations[*location]
// 		if !ok {
// 			locations[*location] = ToCloudyVirtualMachineLocation(location)
// 		}
// 	}
// 	size.Locations = locations

// 	for _, capability := range resource.Capabilities {
// 		switch *capability.Name {
// 		case "vCPUs":
// 			v, err := strconv.ParseInt(*capability.Value, 10, 64)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.CPU = v

// 		case "GPUs":
// 			v, err := strconv.ParseInt(*capability.Value, 10, 64)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.Gpu = v

// 		case "MemoryGB":
// 			v, err := strconv.ParseFloat(*capability.Value, 64)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}
// 			size.RAM = v

// 		case "MaxDataDiskCount":
// 			v, err := strconv.ParseInt(*capability.Value, 10, 64)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.MaxDataDisks = v

// 		case "MaxNetworkInterfaces":
// 			v, err := strconv.ParseInt(*capability.Value, 10, 64)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.MaxNetworkInterfaces = v

// 		case "AcceleratedNetworkingEnabled":
// 			v, err := strconv.ParseBool(*capability.Value)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.AcceleratedNetworking = v

// 		case "PremiumIO":
// 			v, err := strconv.ParseBool(*capability.Value)
// 			if err != nil {
// 				log.ErrorContext(ctx, fmt.Sprintf("capability error: %s %s", *capability.Name, *capability.Value), logging.WithError(err))
// 				continue
// 			}

// 			size.PremiumIo = v

// 		case "MaxResourceVolumeMB",
// 			"OSVhdSizeMB",
// 			"MemoryPreservingMaintenanceSupported",
// 			"HyperVGenerations",
// 			"CpuArchitectureType",
// 			"LowPriorityCapable",
// 			"VMDeploymentTypes",
// 			"vCPUsAvailable",
// 			"ACUs",
// 			"vCPUsPerCore",
// 			"CombinedTempDiskAndCachedIOPS",
// 			"CombinedTempDiskAndCachedReadBytesPerSecond",
// 			"CombinedTempDiskAndCachedWriteBytesPerSecond",
// 			"UncachedDiskIOPS",
// 			"UncachedDiskBytesPerSecond",
// 			"EphemeralOSDiskSupported",
// 			"SupportedEphemeralOSDiskPlacements",
// 			"EncryptionAtHostSupported",
// 			"CapacityReservationSupported",
// 			"CachedDiskBytes",
// 			"UltraSSDAvailable",
// 			"MaxWriteAcceleratorDisksAllowed",
// 			"TrustedLaunchDisabled",
// 			"ParentSize",
// 			"DiskControllerTypes",
// 			"NvmeDiskSizeInMiB",
// 			"NvmeSizePerDiskInMiB",
// 			"HibernationSupported",
// 			"RdmaEnabled":

// 			// These capabilities may be used later
// 			continue

// 		default:
// 			log.InfoContext(ctx, fmt.Sprintf("unhandled capability: %s %s", *capability.Name, *capability.Value))

// 		}

// 	}

// 	return &size
// }

// func ToCloudyVirtualMachineLocation(location *string) *models.VirtualMachineLocation {
// 	return &models.VirtualMachineLocation{
// 		Cloud:  "aws",
// 		Region: *location,
// 	}
// }


// UpdateCloudyVirtualMachine updates the Cloudy VirtualMachine with details from an AWS EC2 instance.
func UpdateCloudyVirtualMachine(vm *models.VirtualMachine, instance *types.Instance) (*models.VirtualMachine, error) {
    if vm.ID == "" {
        vm.ID = *instance.InstanceId
    }

    if vm.State == "" {
        vm.State = string(instance.State.Name)
    }

    if vm.Location == nil {
        vm.Location = &models.VirtualMachineLocation{
            Region: *instance.Placement.AvailabilityZone,
        }
    }

	if vm.Tags == nil {
		vm.Tags = make(map[string]*string)
	}
	
	vm.Tags["AWSID"] = instance.InstanceId

	vm.Name = ""
	for _, tag := range instance.Tags {
		if *tag.Key == "Name" {
			vm.Name = *tag.Value
		}
		
		vm.Tags[*tag.Key] = tag.Value
	}
	if vm.Name == "" {
		return nil, errors.New("Error translating AWS VM to cloudy VM, no 'Name' tag found")
	}

	if instance.State.Name != "" {
		vm.State = string(instance.State.Name)
	} else {
		return nil, fmt.Errorf("VM Create: missing state information from AWS instance")
	}

    if vm.OsBaseImageID == "" && instance.ImageId != nil {
        vm.OsBaseImageID = *instance.ImageId
    }

	// TODO: add disks
    // if len(vm.Disks) == 0 {
    //     vm.Disks = append(vm.Disks, &models.VirtualMachineDisk{
    //         ID: *instance.BlockDeviceMappings[0].Ebs.VolumeId, // Example of attaching an EBS disk
    //     })
    // }

    if vm.Status == "" && instance.State.Name != "" {
        vm.Status = string(instance.State.Name) // Use string directly without dereferencing
    }

	// TODO: update template
    // if vm.Template == nil {
    //     vm.Template = &models.VirtualMachineTemplate{
    //         Name: *instance.InstanceType,
    //     }
    // }

    return vm, nil
}

func (vmm *AwsVirtualMachineManager) FindVMByName(ctx context.Context, name string) (*models.VirtualMachine, error) {
	log := logging.GetLogger(ctx)

	// Filter instances by the "Name" tag
	input := &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "stopping", "stopped"}, // Exclude terminated instances
			},
		},
	}

	output, err := vmm.vmClient.DescribeInstances(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "finding VM by name")
	}

	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			log.InfoContext(ctx, fmt.Sprintf("VM with name '%s' found: %s", name, *instance.InstanceId))
			return ToCloudyVirtualMachine(instance), nil
		}
	}

	log.InfoContext(ctx, fmt.Sprintf("No VM found with name: %s", name))
	return nil, nil
}
