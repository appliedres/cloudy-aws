package cloudyaws

import (
	"context"

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

	// TODO: add AWS permissions check before making requests

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
	return TerminateInstance(ctx, vmc, vmName, wait)
}

func (vmc *AwsEc2Controller) Create(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineConfiguration, error) {
	// TODO: configure credentials on instance from vm config

	cloudy.Info(ctx, "[%s] Starting Create VM", vm.ID)
	err := ValidateConfiguration(ctx, vm)
	if err != nil {
		return vm, err
	}

	// TODO: check if this VM exists first
	

	// Check if NIC already exists
	cloudy.Info(ctx, "[%s] Searching for existing NIC", vm.ID)
	network, err := vmc.GetVmNic(ctx, vm)
	if err != nil {
		cloudy.Info(ctx, "[%s] Error looking for existing NIC: %s", vm.ID, err.Error())
	}

	if network != nil {
		cloudy.Info(ctx, "[%s] Found existing NIC: %s", vm.ID, network.ID)
		vm.PrimaryNetwork = network
	} else {
		// No existing NIC, create one
		cloudy.Info(ctx, "[%s] No existing NIC found, creating one", vm.ID)
		subnetId, err := vmc.FindBestSubnet(ctx, vmc.Config.AvailableSubnets)
		if err != nil {
			return vm, err
		}
		if subnetId == "" {
			return vm, cloudy.Error(ctx, "[%s] no available subnets", vm.ID)
		}

		// Check / Create the Network Interface
		err = vmc.CreateNIC(ctx, vm, subnetId)
		if err != nil {
			return vm, err
		}
	}

	err = CreateInstance(ctx, vmc, vm)
	
	return vm, err
}

func (vmc *AwsEc2Controller) Delete(ctx context.Context, vm *cloudyvm.VirtualMachineConfiguration) (*cloudyvm.VirtualMachineConfiguration, error) {
	cloudy.Info(ctx, "[%s] Starting Delete", vm.ID)
	err := ValidateConfiguration(ctx, vm)
	if err != nil {
		return vm, err
	}

	err = TerminateInstance(ctx, vmc, vm.Name, true)
	if err != nil {
		return vm, err
	}

	err = vmc.DeleteNIC(ctx, vm)
	if err != nil {
		return vm, err
	}

	cloudy.Info(ctx, "[%s] Delete VM Complete", vm.ID)
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
}
