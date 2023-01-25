package cloudyaws

import (
	"context"
	"fmt"

	"github.com/appliedres/cloudy"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/servicequotas"
)

const AwsEc2 = "aws-ec2"

func init() {
	cloudyvm.VmControllers.Register(AwsEc2, &AwsEc2ControllerFactory{})
}

type AwsEc2ControllerConfig struct {
	// TODO: confirm all necessary config items are added
	// AwsCredentials
	// subscriptionId string
	// ResourceGroup  string

	// ??
	// NetworkResourceGroup     string   // From Environment Variable
	// SourceImageGalleryName   string   // From Environment Variable
	// Vnet                     string   // From Environment Variable
	AvailableSubnets []string // From Environment Variable
	// NetworkSecurityGroupName string   // From Environment Variable
	// NetworkSecurityGroupID   string   // From Environment Variable
	// SaltCmd                  string   // From Environment Variable
	// VaultURL                 string

	// DomainControllers []*string // From Environment Variable

	// LogBody bool
}

type AwsEc2Controller struct {
	Quotas    *servicequotas.ServiceQuotas
	Ec2Client *ec2.EC2
	Config    *AwsEc2ControllerConfig
}

type AwsEc2ControllerFactory struct{}

// creates the AWS VM Controller Interface
func (f *AwsEc2ControllerFactory) Create(cfg interface{}) (cloudyvm.VMController, error) {
	awscfg := cfg.(*AwsEc2ControllerConfig)
	if awscfg == nil {
		return nil, cloudy.ErrInvalidConfiguration
	}

	return NewAwsEc2Controller(context.Background(), awscfg)
}

func (f *AwsEc2ControllerFactory) FromEnv(env *cloudy.Environment) (interface{}, error) {
	cfg := &AwsEc2ControllerConfig{}

	// TODO: confirm all necessary config items are added

	// cfg.AWSCredentials = GetAWSCredentialsFromEnv(env)
	// cfg.SubscriptionID = env.Force("AZ_SUBSCRIPTION_ID")
	// cfg.ResourceGroup = env.Force("AZ_RESOURCE_GROUP")
	// cfg.SubscriptionID = env.Force("AZ_SUBSCRIPTION_ID")
	// cfg.SaltCmd = env.Force(("SALT_CMD"))

	// // Not always necessary but needed for creation
	// cfg.NetworkResourceGroup = env.Force("AZ_NETWORK_RESOURCE_GROUP")
	// cfg.SourceImageGalleryName = env.Force("AZ_SOURCE_IMAGE_GALLERY_NAME")
	// cfg.Vnet = env.Force("AZ_VNET")
	// cfg.NetworkSecurityGroupName = env.Force("AZ_NETWORK_SECURITY_GROUP_NAME")
	// cfg.NetworkSecurityGroupID = env.Force("AZ_NETWORK_SECURITY_GROUP_ID")
	// cfg.VaultURL = env.Force("AZ_VAULT_URL")

	// subnets := env.Force("SUBNETS") //SUBNET1,SUBNET2
	// cfg.AvailableSubnets = strings.Split(subnets, ",")

	// domainControllers := strings.Split(env.Force("DOMAIN_CONTROLLERS"), ",") // DC1, DC2
	// for i := range domainControllers {
	// 	cfg.DomainControllers = append(cfg.DomainControllers, &domainControllers[i])
	// }

	// logBody := env.Get("AZ_LOG_BODY")
	// if strings.ToUpper(logBody) == "TRUE" {
	// 	cfg.LogBody = true
	// }

	return cfg, nil
}

func NewAwsEc2Controller(ctx context.Context, config *AwsEc2ControllerConfig) (*AwsEc2Controller, error) {

	// TODO: switch to STS credentials https://docs.aws.amazon.com/sdk-for-go/api/aws/credentials/stscreds/

	sess := session.Must(session.NewSessionWithOptions(session.Options{

		// // Provide SDK Config options, such as Region.
		// Config: aws.Config{
		// 	Region: aws.String("us-east-2"),
		// },

		// Force enable Shared Config support
		SharedConfigState: session.SharedConfigEnable,
	}))

	quotas := servicequotas.New(sess)
	ec2client := ec2.New(sess)

	return &AwsEc2Controller{
		Quotas:    quotas,
		Ec2Client: ec2client,
		Config:    config,
	}, nil
}

// TODO: Validate vmName inputs, ensure it can be stored as 'Name' tag

func (vmc *AwsEc2Controller) ListAll(ctx context.Context) ([]*cloudyvm.VirtualMachineStatus, error) {
	return ListAllInstances(ctx, vmc)
}

func (vmc *AwsEc2Controller) ListWithTag(ctx context.Context, tag string) ([]*cloudyvm.VirtualMachineStatus, error) {
	return ListInstancesWithTag(ctx, vmc, tag)
}

func (vmc *AwsEc2Controller) Status(ctx context.Context, vmName string) (*cloudyvm.VirtualMachineStatus, error) {
	return InstanceStatusByVmName(ctx, vmc, vmName)
}

func (vmc *AwsEc2Controller) SetState(ctx context.Context, state cloudyvm.VirtualMachineAction, vmName string, wait bool) (*cloudyvm.VirtualMachineStatus, error) {
	return SetInstanceState(ctx, vmc, state, vmName, wait)
}

func (vmc *AwsEc2Controller) Start(ctx context.Context, vmName string, wait bool) error {
	return StartInstance(ctx, vmc, vmName, wait)
}

func (vmc *AwsEc2Controller) Stop(ctx context.Context, vmName string, wait bool) error {
	return StopInstance(ctx, vmc, vmName, wait)
}

func (vmc *AwsEc2Controller) Terminate(ctx context.Context, vmName string, wait bool) error {
	return TerminateVmInstance(ctx, vmc, vmName, wait)
}

func (vmc *AwsEc2Controller) Create(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineConfiguration, error) {
	cloudy.Info(ctx, "[%s] Starting Create", vm.ID)
	err := ValidateConfiguration(ctx, vm)
	if err != nil {
		return vm, err
	}

	// Check if NIC already exists
	cloudy.Info(ctx, "[%s] Starting GetNIC", vm.ID)
	network, err := vmc.GetVmNic(ctx, vm)
	if err != nil {
		cloudy.Info(ctx, "[%s] Error looking for NIC: %s", vm.ID, err.Error())
	}

	if network != nil {
		cloudy.Info(ctx, "[%s] Using existing NIC: %s", vm.ID, network.ID)
		vm.PrimaryNetwork = network
	} else {
		// No existing NIC, create one
		cloudy.Info(ctx, "[%s] No NIC found, creating one", vm.ID)
		subnetId, err := vmc.FindBestSubnet(ctx, vmc.Config.AvailableSubnets)
		if err != nil {
			return vm, err
		}
		if subnetId == "" {
			return vm, fmt.Errorf("[%s] no available subnets", vm.ID)
		}

		// Check / Create the Network Interface
		err = vmc.CreateNIC(ctx, vm, subnetId)
		if err != nil {
			return vm, err
		}
	}

	cloudy.Info(ctx, "[%s] Starting CreateVirtualMachine", vm.ID)

	instanceOptions := &ec2.RunInstancesInput{
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sdh"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize: aws.Int64(100),
				},
			},
		},
		ImageId:      aws.String("ami-0ab0629dba5ae551d"), // TODO: make dynamic, this is hardcoded for Ubuntu Server 22.04
		InstanceType: aws.String(vm.SizeRequest.SpecificSize),  // TODO: use Size.Name or SizeRequest.SpecificSize?
		// KeyName:      aws.String("my-key-pair"),
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
		// SecurityGroupIds: []*string{
		// 	aws.String("sg-1a2b3c4d"),
		// },
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(vm.Name),
					},
				},
			},
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:        aws.Int64(0), // Primary NIC
				NetworkInterfaceId: &vm.PrimaryNetwork.ID,
			},
		},
	}

	runResult, err := vmc.Ec2Client.RunInstances(instanceOptions)
	if err != nil {
		fmt.Println("Could not create instance: ", err)
		// TODO: delete created NIC, EBS, etc on create failure
		return vm, err
	}
	// TODO: do we need to store instance ID?

	fmt.Println("Created instance", *runResult.Instances[0].InstanceId)
	return vm, err
}

func (vmc *AwsEc2Controller) Delete(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineConfiguration, error) {
	cloudy.Info(ctx, "[%s] Starting Delete", vm.ID)
	err := ValidateConfiguration(ctx, vm)
	if err != nil {
		return vm, err
	}

	err = TerminateVmInstance(ctx, vmc, vm.Name, true)
	if err != nil {
		return vm, err
	}

	err = vmc.DeleteNIC(ctx, vm)
	if err != nil {
		return vm, err
	}

	cloudy.Info(ctx, "[%s] Deleted VM", vm.ID)
	return vm, err
}

func (vmc *AwsEc2Controller) GetLimits(ctx context.Context) ([]*cloudyvm.VirtualMachineLimit, error) {
	// TODO: Look up current usage
	// TOOD: Match quota name or code to ec2 size

	var rtn []*cloudyvm.VirtualMachineLimit

	out, err := vmc.Quotas.ListServiceQuotas(&servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String("ec2"),
	})
	if err != nil {
		return nil, err
	}

	for {
		for _, q := range out.Quotas {
			rtn = append(rtn, &cloudyvm.VirtualMachineLimit{
				Name:  *q.QuotaName,
				Limit: int(*q.Value),
			})
		}

		if out.NextToken != nil {
			out, err = vmc.Quotas.ListServiceQuotas(&servicequotas.ListServiceQuotasInput{
				ServiceCode: aws.String("ec2"),
				NextToken:   out.NextToken,
			})

			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return rtn, nil
}

func (vmc *AwsEc2Controller) GetVMSizes(ctx context.Context) (map[string]*cloudyvm.VmSize, error) {
	// TODO: GetVMSizes

	resp, err := vmc.Ec2Client.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{})
	if err != nil {
		return nil, err
	}

	rtn := make(map[string]*cloudyvm.VmSize)
	for {
		for _, offer := range resp.InstanceTypes {
			size := &cloudyvm.VmSize{}

			size.Name = *offer.InstanceType

			rtn[size.Name] = size

			if resp.NextToken != nil {
				resp, err = vmc.Ec2Client.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{
					NextToken: resp.NextToken,
				})
				if err != nil {
					return nil, err
				}
			}
		}

	}

	return rtn, nil

	return nil, nil
}
