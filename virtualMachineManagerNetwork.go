package cloudyaws

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/appliedres/cloudy/logging"
	"github.com/appliedres/cloudy/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/pkg/errors"
)

func (vmm *AwsVirtualMachineManager) GetAllNics(ctx context.Context) ([]*models.VirtualMachineNic, error) {
	nics := []*models.VirtualMachineNic{}

	input := &ec2.DescribeNetworkInterfacesInput{}
	paginator := ec2.NewDescribeNetworkInterfacesPaginator(vmm.vmClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve network interfaces: %w", err)
		}

		for _, eni := range page.NetworkInterfaces {
			var name string
			for _, tag := range eni.TagSet {
				if aws.ToString(tag.Key) == "Name" {
					name = aws.ToString(tag.Value)
					break
				}
			}

			nic := &models.VirtualMachineNic{
				ID:        aws.ToString(eni.NetworkInterfaceId),
				Name:      name,
				PrivateIP: aws.ToString(eni.PrivateIpAddress),
			}
			nics = append(nics, nic)
		}
	}

	return nics, nil
}

func (vmm *AwsVirtualMachineManager) GetNics(ctx context.Context, vmId string) ([]*models.VirtualMachineNic, error) {
	nics := []*models.VirtualMachineNic{}

	allNics, err := vmm.GetAllNics(ctx)
	if err != nil {
		return nil, err
	}

	for _, nic := range allNics {
		if strings.Contains(nic.Name, vmId) {
			nics = append(nics, nic)
		}
	}

	return nics, nil
}

func (vmm *AwsVirtualMachineManager) CreateNic(ctx context.Context, vm *models.VirtualMachine) (*models.VirtualMachineNic, error) {
	log := logging.GetLogger(ctx)

	subnetID, err := vmm.findBestSubnet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find the best subnet: %w", err)
	}

	nicName := fmt.Sprintf("%s-nic-primary", vm.ID)

	// TODO: DNS servers cannot be set at the NIC level in AWS
	// dnsServers := []*string{}
	// if strings.EqualFold(vm.Template.OperatingSystem, "windows") {
	// 	dnsServers = vmm.config.DomainControllers
	// }

	input := &ec2.CreateNetworkInterfaceInput{
		SubnetId: aws.String(subnetID),
		Description: aws.String(fmt.Sprintf("Primary NIC for VM: %s", vm.ID)),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeNetworkInterface,
				Tags: []types.Tag{
					{Key: aws.String("Name"), Value: aws.String(nicName)},
				},
			},
		},
	}

	output, err := vmm.vmClient.CreateNetworkInterface(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create network interface: %w", err)
	}

	nic := &models.VirtualMachineNic{
		ID:        aws.ToString(output.NetworkInterface.NetworkInterfaceId),
		Name:      nicName,
		PrivateIP: aws.ToString(output.NetworkInterface.PrivateIpAddress),
	}

	log.InfoContext(ctx, fmt.Sprintf("Created new NIC: %s", nic.ID))

	return nic, nil
}


func (vmm *AwsVirtualMachineManager) DeleteNics(ctx context.Context, nics []*models.VirtualMachineNic) error {

	for _, nic := range nics {
		err := vmm.DeleteNic(ctx, nic)
		if err != nil {
			return errors.Wrap(err, "DeleteNics")
		}
	}

	return nil
}

func (vmm *AwsVirtualMachineManager) DeleteNic(ctx context.Context, nic *models.VirtualMachineNic) error {
	log := logging.GetLogger(ctx)
	log.InfoContext(ctx, fmt.Sprintf("DeleteNic starting: %s", nic.Name))

	_, err := vmm.vmClient.DeleteNetworkInterface(ctx, &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(nic.ID),
	})
	if err != nil {
		return fmt.Errorf("DeleteNic: failed to delete NIC %s: %w", nic.Name, err)
	}

	log.InfoContext(ctx, fmt.Sprintf("DeleteNic completed: %s", nic.Name))
	return nil
}


func (vmm *AwsVirtualMachineManager) findBestSubnet(ctx context.Context) (string, error) {
	log := logging.GetLogger(ctx)

	if len(vmm.config.SubnetIds) == 0 {
		return "", fmt.Errorf("no subnets specified in the configuration")
	}

	bestSubnetId := ""
	bestSubnetCount := 0

	for _, subnetId := range vmm.config.SubnetIds {
		subnetCount, err := vmm.getSubnetAvailableIps(ctx, subnetId)
		if err != nil {
			log.ErrorContext(ctx, fmt.Sprintf("error counting ips in subnet: %s", subnetId), logging.WithError(err))
			continue
		}

		if bestSubnetId == "" || subnetCount > bestSubnetCount {
			bestSubnetId = subnetId
			bestSubnetCount = subnetCount
		}
	}

	if bestSubnetId == "" {
		return "", fmt.Errorf("could not find any suitable subnets in subnets %s", vmm.config.SubnetIds)
	}

	return bestSubnetId, nil
}

func (vmm *AwsVirtualMachineManager) getSubnetAvailableIps(ctx context.Context, subnetId string) (int, error) {
	res, err := vmm.vmClient.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetId},
	})
	if err != nil {
		return 0, fmt.Errorf("getSubnetAvailableIps: failed to find subnet %s: %w", subnetId, err)
	}

	if len(res.Subnets) == 0 {
		return 0, fmt.Errorf("getSubnetAvailableIps: no subnets found for ID %s", subnetId)
	}

	subnet := res.Subnets[0]

	// Get the CIDR block
	if subnet.CidrBlock == nil {
		return 0, fmt.Errorf("getSubnetAvailableIps: CIDR block not found for subnet %s", subnetId)
	}

	cidrParts := strings.Split(*subnet.CidrBlock, "/")
	if len(cidrParts) != 2 {
		return 0, fmt.Errorf("getSubnetAvailableIps: invalid CIDR block %s", *subnet.CidrBlock)
	}

	subnetMask, err := strconv.Atoi(cidrParts[1])
	if err != nil {
		return 0, fmt.Errorf("getSubnetAvailableIps: invalid subnet mask %s: %w", cidrParts[1], err)
	}

	// Calculate total IPs from the CIDR block
	totalIPs := int(math.Pow(2, float64(32-subnetMask)))

	// Subtract reserved and in-use IPs
	reservedIPs := 5 // AWS reserves 5 IPs in each subnet
	inUseIPs := int(*subnet.AvailableIpAddressCount)
	availableIPs := totalIPs - reservedIPs - inUseIPs

	if availableIPs < 0 {
		availableIPs = 0
	}

	return availableIPs, nil
}
