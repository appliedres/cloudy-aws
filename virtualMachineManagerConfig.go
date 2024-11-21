package cloudyaws

type VirtualMachineManagerConfig struct {
	DomainControllers []*string

	SubnetIds []string

	VpcID string
}
