package cloudyaws

import (
	"context"
	"fmt"

	"github.com/appliedres/cloudy"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// translates AWS create NIC output to cloudy NIC
func (vmc *AwsEc2Controller) translateCreateNICoutput(output *ec2.CreateNetworkInterfaceOutput) (*cloudyvm.VirtualMachineNetwork, error) {
	var err error

	cloudyNIC, err := vmc.translateAwsNicToCloudyNic(output.NetworkInterface)

	return cloudyNIC, err
}

// given an EC2 Network Interface, gets NIC name from 'Name' Tag value, otherwise returns an empty string
func (vmc *AwsEc2Controller) getNicNameTagValue(awsNIC *ec2.NetworkInterface) (string, error) {
	var err error

	name := ""
	for _, t := range awsNIC.TagSet {
		if *t.Key == "Name" {
			name = *t.Value
		}
	}

	return name, err
}

// translates AWS EC2 NetworkInterface to Cloudy VirtualMachineNetwork
func (vmc *AwsEc2Controller) translateAwsNicToCloudyNic(awsNIC *ec2.NetworkInterface) (*cloudyvm.VirtualMachineNetwork, error) {
	var err error

	name, err := vmc.getNicNameTagValue(awsNIC)
	if err != nil {
		return nil, err
	}

	cloudyNIC := &cloudyvm.VirtualMachineNetwork{
		ID:        *awsNIC.NetworkInterfaceId,
		Name:      name,
		PrivateIP: *awsNIC.PrivateIpAddress,
	}

	return cloudyNIC, err
}

// translates AWS DescribeNetworkInterfacesOutput to a slice of Cloudy VirtualMachineNetworks
func (vmc *AwsEc2Controller) translateDescribeNICsOutput(output *ec2.DescribeNetworkInterfacesOutput) ([]*cloudyvm.VirtualMachineNetwork, error) {
	var err error
	var cloudyNICs []*cloudyvm.VirtualMachineNetwork

	for _, awsNIC := range output.NetworkInterfaces {
		cloudyNIC, err := vmc.translateAwsNicToCloudyNic(awsNIC)
		if err != nil {
			return nil, err
		}

		cloudyNICs = append(cloudyNICs, cloudyNIC)
	}
	return cloudyNICs, err
}

// Calls getNICs with an empty filter to get all NICs
func (vmc *AwsEc2Controller) GetAllNICs(ctx context.Context) ([]*cloudyvm.VirtualMachineNetwork, error) {
	return vmc.getNICs(ctx, &ec2.DescribeNetworkInterfacesInput{})
}

// using a given input filter, get matching NICs
// returns nil if no matching NICs are found
func (vmc *AwsEc2Controller) getNICs(ctx context.Context, input *ec2.DescribeNetworkInterfacesInput) ([]*cloudyvm.VirtualMachineNetwork, error) {
	var err error
	var cloudyNICs []*cloudyvm.VirtualMachineNetwork

	desNicOut, err := vmc.Ec2Client.DescribeNetworkInterfaces(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidNetworkInterfaceID.NotFound":
				return nil, nil
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

	cloudyNICs, err = vmc.translateDescribeNICsOutput(desNicOut)

	return cloudyNICs, err
}

// Finds EC2 Network Interfaces that have a matching name and returns a list of them in cloudy format
func (vmc *AwsEc2Controller) FindNICsByName(ctx context.Context, nicName string) ([]*cloudyvm.VirtualMachineNetwork, error) {

	input := &ec2.DescribeNetworkInterfacesInput{
		// NetworkInterfaceIds: []*string{
		// 	aws.String("eni-e5aa89a3"),
		// },
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(nicName),
				},
			},
		},
	}

	return vmc.getNICs(ctx, input)
}

// Finds an EC2 Network Interface that has a matching ID and returns it in cloudy format.
// Returns nil if no NIC found
func (vmc *AwsEc2Controller) FindNicByID(ctx context.Context, id string) (*cloudyvm.VirtualMachineNetwork, error) {

	cloudy.Info(ctx, "Searching for NIC with ID = '%s'", id)

	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{
			aws.String(id),
		},
	}

	nics, err := vmc.getNICs(ctx, input)
	if err != nil {
		return nil, err
	} else if nics == nil {
		cloudy.Info(ctx, "NIC not found with ID = '%s'", id)
		return nil, err
	}

	cloudy.Info(ctx, "Successfully found NIC with ID = '%s'", id)
	return nics[0], err
}

// Finds the primary EC2 Network Interface for a given VM and returns it in cloudy format
func (vmc *AwsEc2Controller) GetVmNic(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineNetwork, error) {
	nicName := vm.Name + "-nic-primary"
	nics, err := vmc.FindNICsByName(ctx, nicName)
	if err != nil {
		return nil, err
	} else if nics == nil {
		return nil, cloudy.Error(ctx, "zero NICs found with name '%s'", nicName)
	} else if len(nics) > 1 {
		return nil, cloudy.Error(ctx, "multiple NICs found with name '%s'", nicName)
	}

	// TODO: should the NIC name be verified here?

	return nics[0], err
}

func (vmc *AwsEc2Controller) CreateNIC(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration, subnetId string) error {
	// TODO: check that subnet ID exists

	cloudy.Info(ctx, "[%s] Starting CreateNIC", vm.ID)

	if vm.PrimaryNetwork != nil {
		return cloudy.Error(ctx, "[%s] VM config already has a NIC", vm.ID)
	}

	nicName := fmt.Sprintf("%v-nic-primary", vm.ID)
	// region := vmc.Config.Region
	// rg := vmc.Config.NetworkResourceGroup

	// verify no NICs exist with this name
	matchingNICs, err := vmc.FindNICsByName(ctx, nicName)
	if err != nil {
		return err
	}

	switch s := len(matchingNICs); {
	case s > 1:
		return cloudy.Error(ctx, "multiple matching NICs (%d) already exist with name '%s'", s, nicName)
	case s == 1:
		return cloudy.Error(ctx, "a NIC already exists with name '%s'", nicName)
	case s != 0:
		return cloudy.Error(ctx, "invalid number of NICs (%d) found for name '%s'", s, nicName)
	default: // proceed with NIC creation
	}

	// create NIC
	input := &ec2.CreateNetworkInterfaceInput{
		Description: aws.String(fmt.Sprintf("primary network interface created by cloudy-aws for vm ID [%s]", vm.ID)),
		// Groups: []*string{
		// 	aws.String("sg-903004f8"),
		// },
		// PrivateIpAddress: aws.String("10.0.2.17"),
		SubnetId: aws.String(subnetId),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("network-interface"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(nicName),
					},
				},
			},
		},
	}

	result, err := vmc.Ec2Client.CreateNetworkInterface(input)
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
		return err
	}

	// translate NIC to cloudy format
	nic, err := vmc.translateCreateNICoutput(result)
	if err != nil {
		return err
	}

	// create successful, save this NIC to VM config
	vm.PrimaryNetwork = nic

	cloudy.Info(ctx, "[%s] CreateNIC successful\n\tName = %s", vm.ID, vm.PrimaryNetwork.Name)

	return nil
}

func (vmc *AwsEc2Controller) DeleteNIC(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) error {
	cloudy.Info(ctx, "[%s] Starting DeleteNIC", vm.ID)

	// if vm.PrimaryNetwork == nil || vm.PrimaryNetwork.ID == "" {
	// 	cloudy.Info(ctx, "[%s] No primary NIC found, nothing to delete", vm.ID)
	// 	return nil // no NIC attached to this VM
	// }
	// nicID := vm.PrimaryNetwork.ID
	// region := vmc.Config.Region
	// rg := vmc.Config.NetworkResourceGroup

	// verify target NIC exists
	// nic, err := vmc.FindNicByID(ctx, nicID)
	// if err != nil {
	// 	return err
	// } else if nic == nil {
	// 	return cloudy.Error(ctx, "[%s] could not delete NIC, no matching NIC found with ID = '%s'", vm.ID, nicID)
	// }

	nicName := fmt.Sprintf("%s-nic-primary", vm.Name)
	nics, err := vmc.FindNICsByName(ctx, nicName)
	if err != nil {
		return err
	}
	nic := nics[0]

	// TODO: check that NIC is detached prior to deletion

	cloudy.Info(ctx, "[%s] NIC found, deleting...", vm.ID)

	input := &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(nic.ID),
	}

	_, err = vmc.Ec2Client.DeleteNetworkInterface(input)
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
		return err
	}

	// verify NIC deletion
	nics, err = vmc.FindNICsByName(ctx, nicName)
	if err != nil {
		return err
	}
	if (len(nics) > 0) {
		return cloudy.Error(ctx, "[%s] NIC delete failed with name = '%s'", vm.ID, nicName)
	}

	vm.PrimaryNetwork = nil
	cloudy.Info(ctx, "[%s] DeleteNIC successful", vm.ID)

	return nil
}
