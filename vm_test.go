package cloudyaws

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/stretchr/testify/assert"
)

func TestListAll(t *testing.T) {
	fmt.Println("Testing ListAllInstances")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})

	assert.Nil(t, err)

	all, err := vmc.ListAll(ctx)
	assert.Nil(t, err)

	assert.NotNil(t, all)
	fmt.Printf("Found %d instances:\n", len(all))
	for _, vm := range all {
		fmt.Printf("NAME:%v -- ID:%v -- STATE:%s\n", vm.Name, vm.ID, vm.PowerState)
	}
}

func TestStatus(t *testing.T) {
	name := "manual-name"
	fmt.Printf("Testing TestStatus\nName=%s\n", name)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})

	assert.Nil(t, err)

	status, err := vmc.Status(ctx, name)
	assert.Nil(t, err)

	fmt.Printf("Instance status for name '%s':\n%v", name, status)
}


func TestStop(t *testing.T) {
	name := "manual-name"
	fmt.Printf("Testing TestListWithName\nName=%s\n", name)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})

	assert.Nil(t, err)

	err = vmc.Stop(ctx, name, false)
	assert.Nil(t, err)

	// fmt.Printf("Instance status for name '%s':\n%v", name, status)
}

func TestStart(t *testing.T) {
	name := "manual-name"
	fmt.Printf("Testing TestListWithName\nName=%s\n", name)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})

	assert.Nil(t, err)

	err = vmc.Start(ctx, name, false)
	assert.Nil(t, err)

	// fmt.Printf("Instance status for name '%s':\n%v", name, status)
}

func TestTerminate(t *testing.T) {
	name := "manual-name"
	fmt.Printf("Testing TestTerminate\nvm name=%s\n", name)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})

	assert.Nil(t, err)

	err = vmc.Terminate(ctx, name, false)
	assert.Nil(t, err)

	// fmt.Printf("Instance status for name '%s':\n%v", name, status)
}


func TestCreateAndDeleteVM(t *testing.T) {
	time_ms := time.Now().UnixNano()/1000000
	vmID := fmt.Sprintf("uvm-gotest_T%d", time_ms)
	fmt.Printf("Testing TestCreateVM with VM name = %s\n", vmID)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	subnets := strings.Split(os.Getenv("SUBNETS"), ",") // TODO: use cloudy environment

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{
		AvailableSubnets:         subnets,
	})
	assert.Nil(t, err)

	vmConfig := &cloudyvm.VirtualMachineConfiguration{
		ID:   vmID,
		Name: "testName-" + vmID,
		Size: &cloudyvm.VmSize{
			Name: "t2.micro",
		},
		SizeRequest: &cloudyvm.VmSizeRequest{
			SpecificSize: "t2.micro",
		},
		OSType:       "linux",
		Image:        "canonical::ubuntuserver::19.04",
		ImageVersion: "19.04.202001220",
		Credientials: cloudyvm.Credientials{
			AdminUser:     "salt",
			AdminPassword: "TestPassword12#$",
			// SSHKey:        sshPublicKey,
		},
	}

	_, err = vmc.Create(ctx, vmConfig)
	assert.Nil(t, err)

	status, err := InstanceStatusByVmName(ctx, vmc, vmConfig.Name)
	assert.Nil(t, err)
	assert.NotNil(t, status)

	_, err = vmc.Delete(ctx, vmConfig)
	assert.Nil(t, err)

	status, err = InstanceStatusByVmName(ctx, vmc, vmConfig.Name)
	assert.Nil(t, err)
	// TODO: status could be terminated
	assert.Equal(t, status.PowerState, "shutting-down")
}
