package cloudyaws

import (
	"context"
	"strings"
	"regexp"

	"github.com/appliedres/cloudy"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/servicequotas"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const Aws = "aws"

func init() {
	cloudyvm.VmControllers.Register(Aws, &AwsEc2ControllerFactory{})
}

type AwsEc2ControllerConfig struct {
	// TODO: confirm all necessary config items are added
	AwsCredentials

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
	QuotasClient    *servicequotas.ServiceQuotas
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

	cfg.AwsCredentials = GetAwsCredentialsFromEnv(env)

	// cfg.SaltCmd = env.Force(("SALT_CMD"))

	// // Not always necessary but needed for creation
	// cfg.NetworkResourceGroup = env.Force("AZ_NETWORK_RESOURCE_GROUP")
	// cfg.SourceImageGalleryName = env.Force("AZ_SOURCE_IMAGE_GALLERY_NAME")
	// cfg.Vnet = env.Force("AZ_VNET")
	// cfg.NetworkSecurityGroupName = env.Force("AZ_NETWORK_SECURITY_GROUP_NAME")
	// cfg.NetworkSecurityGroupID = env.Force("AZ_NETWORK_SECURITY_GROUP_ID")
	// cfg.VaultURL = env.Force("AZ_VAULT_URL")

	subnets := env.Force("SUBNETS") //SUBNET1,SUBNET2
	cfg.AvailableSubnets = strings.Split(subnets, ",")

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
    sess, err := session.NewSessionWithOptions(session.Options{
        Config: aws.Config{
			Credentials: credentials.NewStaticCredentials(
				config.AwsCredentials.AccessKeyID, 
				config.AwsCredentials.SecretAccessKey, 
				"",
			),
			Region:      aws.String(config.AwsCredentials.Region),
        },
    })
	if err != nil {
		return nil, err
	}

	// TODO: add AWS permissions check before making requests

	quotas := servicequotas.New(sess)
	ec2client := ec2.New(sess)

	return &AwsEc2Controller{
		QuotasClient:    quotas,
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
	// return InstanceStatusByVmName(ctx, vmc, vmName)
	return nil, nil
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
	cloudy.Info(ctx, "[%s] Start VM Delete", vm.ID)

	err := TerminateInstance(ctx, vmc, vm.Name, true)
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

// lookup of quota codes for all supported instance families
// aws does not provide a way to determine quota from instanceType directly, so this is necessary
var supportedInstanceTypeQuotaCodes = map[string]string{
	"a": "L-1216C47A", 
	"c": "L-1216C47A",
	"d": "L-1216C47A",
	"h": "L-1216C47A",
	"i": "L-1216C47A",
	"m": "L-1216C47A",
	"r": "L-1216C47A",
	"t": "L-1216C47A",
	"z": "L-1216C47A",
	"f": "L-74FC7D96",
	"p": "L-417A185B",

}

// For all instance types, this retreives the number of currently running instances and the quota remaining
func (vmc *AwsEc2Controller) GetLimits(ctx context.Context) ([]*cloudyvm.VirtualMachineLimit, error) {
	var rtn []*cloudyvm.VirtualMachineLimit

	allInstanceTypes, err := vmc.GetVMSizes(ctx)
	if err != nil {
		return nil, err
	}
	cloudy.Info(ctx, "Found %d available instances types", len(allInstanceTypes))

	// get all running instances in one call, then relate those counts to available types
	running, err := vmc.Ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	totalRunning := len(running.Reservations)
	cloudy.Info(ctx, "Found %d running instances", totalRunning)

	// count running instances by type
	runningByType := make(map[string]int)
	for _, reservation := range running.Reservations {
		for _, instance := range reservation.Instances {
			cloudy.Info(ctx, "found running instance of type:%s", *instance.InstanceType)
			runningByType[*instance.InstanceType] += 1
		}
	}

	// lookup quotas and store them by quota code
	quotaValsByCode := make(map[string]int)
	resp, err := vmc.QuotasClient.ListServiceQuotas(&servicequotas.ListServiceQuotasInput{
		ServiceCode:     aws.String("ec2"),
	})
	if err != nil {
		return nil, err
	}

	for _, quota := range resp.Quotas {
		quotaValsByCode[*quota.QuotaCode] = int(*quota.Value)
	}

	// match number of running and quota to each instance type
	for _, instanceType := range allInstanceTypes {
		numRunning := runningByType[instanceType.Name]
		// cloudy.Info(ctx, "%s:%d", instanceType.Name, numRunning)

		// regex to decompose instanceType into family, gen, extra and size
		var re = regexp.MustCompile(`(\w+|u-\w+)(\d)([a-z]*).(\w+)$`)
		matches := re.FindStringSubmatch(instanceType.Name)
		if len(matches) < 1 {
			return nil, cloudy.Error(ctx, "regex match failed on instanceType: [%s]", instanceType.Name)
		}

		// regexMatch := matches[0]
		family := matches[1]
		// gen := matches[2]
		// extra := matches[3]
		// size := matches[4]
		// cloudy.Info(ctx, "instanceType regex matched:[%s]. family:%s, gen:%s, extra:%s, size:%s", regexMatch, family, gen, extra, size)

		// determine quota code from family
		quotaCode := supportedInstanceTypeQuotaCodes[family]
		if quotaCode == "" {
			cloudy.Info(ctx, "ignoring instance type [%s], ec2 family [%s] is not in supported list", instanceType.Name, family)
			continue;
		}

		quotaValue := quotaValsByCode[quotaCode]
		
		// cloudy.Info(ctx, "instanceType:%s, running:%d, quota:%d", instanceType.Name, numRunning, quotaValue)
		rtn = append(rtn, &cloudyvm.VirtualMachineLimit{
			Name:    instanceType.Name,
			Current: numRunning,
			Limit:   quotaValue,
		})
	}

	return rtn, nil
}

// retrieves all available ec2 instance types
func (vmc *AwsEc2Controller) GetVMSizes(ctx context.Context) (map[string]*cloudyvm.VmSize, error) {
	// TODO: further limit this to only ec2 instance types
	
	cloudy.Info(ctx, "Getting VM Sizes")

	// filter: supported-usage-class = on-demand
	resp, err := vmc.Ec2Client.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("supported-usage-class"),
				Values: []*string{
					aws.String("on-demand"),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	rtn := make(map[string]*cloudyvm.VmSize)
	for {
		for _, offer := range resp.InstanceTypes {
			size := &cloudyvm.VmSize{}

			size.Name = *offer.InstanceType

			rtn[size.Name] = size
			// cloudy.Info(ctx, "%s", size.Name)
		}

		if resp.NextToken != nil {
			resp, err = vmc.Ec2Client.DescribeInstanceTypes(&ec2.DescribeInstanceTypesInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("supported-usage-class"),
						Values: []*string{
							aws.String("on-demand"),
						},
					},
				},
				NextToken: resp.NextToken,
			})
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}


	cloudy.Info(ctx, "GetVMSizes: found %d instance types", len(rtn))
	return rtn, nil
}
