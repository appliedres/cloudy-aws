package cloudyaws

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/stretchr/testify/assert"
)

var vmID string = "uvm-gotest"

func TestListAll(t *testing.T) {
	fmt.Println("TEST: ListAllInstances")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	all, err := vmc.ListAll(ctx)
	assert.Nil(t, err)

	assert.NotNil(t, all)
	fmt.Printf("Found %d instances:\n", len(all))
	for _, vm := range all {
		fmt.Printf("NAME:%v -- ID:%v -- STATE:%s\n", vm.Name, vm.ID, vm.PowerState)
	}
}

func TestCreateVM(t *testing.T) {
	fmt.Printf("TEST: Create VM with name = %s\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	subnets := strings.Split(cloudy.ForceEnv("VMC_SUBNETS", ""), ",")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AvailableSubnets:         subnets,
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	vmConfig := &cloudyvm.VirtualMachineConfiguration{
		ID:   vmID,
		Name: vmID,
		Size: &cloudyvm.VmSize{
			Name: "t2.micro",
		},
		SizeRequest: &cloudyvm.VmSizeRequest{
			SpecificSize: "t2.micro",
		},
		OSType:       "linux",
		Image:        "canonical::ubuntuserver::19.04",
		ImageVersion: "19.04.202001220",
	}

	_, err = vmc.Create(ctx, vmConfig)
	assert.Nil(t, err)

	// status, err := InstanceStatusByVmName(ctx, vmc, vmConfig.Name)
	// assert.Nil(t, err)
	// assert.NotNil(t, status)
}

func TestStop(t *testing.T) {
	fmt.Printf("TEST: Stop VM name='%s'\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	err = vmc.Stop(ctx, vmID, false)
	assert.Nil(t, err)
}

func TestStatus(t *testing.T) {
	fmt.Printf("TEST: Status for VM name=%s\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	status, err := vmc.Status(ctx, vmID)
	assert.Nil(t, err)

	fmt.Printf("Instance status for name '%s': %v\n", vmID, status.PowerState)
}

func TestStart(t *testing.T) {
	fmt.Printf("TEST: Start VM name='%s'\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	err = vmc.Start(ctx, vmID, false)
	assert.Nil(t, err)
}

func TestDeleteVM(t *testing.T) {
	fmt.Printf("TEST: Delete VM with name = %s\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	subnets := strings.Split(os.Getenv("SUBNETS"), ",")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AvailableSubnets:         subnets,
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	vmConfig := &cloudyvm.VirtualMachineConfiguration{
		ID:   vmID,
		Name: vmID,
		Size: &cloudyvm.VmSize{
			Name: "t2.micro",
		},
		SizeRequest: &cloudyvm.VmSizeRequest{
			SpecificSize: "t2.micro",
		},
		OSType:       "linux",
		Image:        "canonical::ubuntuserver::19.04",
		ImageVersion: "19.04.202001220",
	}

	_, err = vmc.Delete(ctx, vmConfig)
	assert.Nil(t, err)

	// status, err := InstanceStatusByVmName(ctx, vmc, vmConfig.Name)
	// assert.Nil(t, err)
	// assert.Equal(t, status.PowerState, "terminated")
}

func TestGetVMSizes(t *testing.T) {
	fmt.Printf("TEST: GetVMSizes")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	subnets := strings.Split(os.Getenv("SUBNETS"), ",")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AvailableSubnets:         subnets,
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	sizes, err := vmc.GetVMSizes(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, sizes)

}

func TestGetLimits(t *testing.T) {
	fmt.Printf("TEST: GetLimits")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	subnets := strings.Split(os.Getenv("SUBNETS"), ",")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AvailableSubnets:         subnets,
		AwsCredentials: 		AwsCredentials{
			Region:       cloudy.ForceEnv("AWS_REGION", ""),
			AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
	})
	assert.Nil(t, err)

	limits, err := vmc.GetLimits(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, limits)

}

