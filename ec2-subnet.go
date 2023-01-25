package cloudyaws

import (
	"context"
	"fmt"

	"github.com/appliedres/cloudy"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Returns the first subnet found with IP availability out of availableSubnets
func (vmc *AwsEc2Controller) FindBestSubnet(ctx context.Context, availableSubnets []string) (string, error) {
	cloudy.Info(ctx, "Starting FindBestSubnet")
	
	if len(availableSubnets) == 0 {
		return "", cloudy.Error(ctx, "No subnets available")
	}
	
	for _, subnetID := range availableSubnets {
		available, err := vmc.GetAvailableIPs(ctx, subnetID)

		if err != nil {
			return "", err
		}
		cloudy.Info(ctx, "Available IPs for subnetID %s: %d", subnetID, available)

		if available > 0 {
			cloudy.Info(ctx, "Selected subnet with availability: ID='%s'", subnetID)
			return subnetID, nil
		}
	}

	// No available subnets
	cloudy.Info(ctx, "No subnets with IP availability")
	return "", nil
}

// Retrieves the number of available IPs in a subnet
func (vmc *AwsEc2Controller) GetAvailableIPs(ctx context.Context, subnetID string) (int, error) {
	// TODO: validate subnetID

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("subnet-id"),
				Values: []*string{
					aws.String(subnetID),
				},
			},
		},
	}
	
	foundSubnets, err := vmc.Ec2Client.DescribeSubnets(input)
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
		return 0, err
	}

	switch l := len(foundSubnets.Subnets); {
	case l > 1: return 0, fmt.Errorf("multiple subnets (%d) found with ID '%s'", l, subnetID)
	case l == 0: return 0, fmt.Errorf("could not find subnet with ID '%s'", subnetID)
	case l != 1: return 0, fmt.Errorf("invalid number of subnets (%d) found for ID '%s'", l, subnetID)
	default:
	}

	subnet := foundSubnets.Subnets[0]
	
	availableIPs64 := *subnet.AvailableIpAddressCount

	availableIPs := int(availableIPs64)
	if int64(availableIPs) != availableIPs64 {
		return availableIPs, fmt.Errorf("error casting int64 to int")
	} 

	return availableIPs, err
}