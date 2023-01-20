package cloudyaws

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"
	cloudyvm "github.com/appliedres/cloudy/vm"

	"github.com/stretchr/testify/assert"
)


func TestGetAllNICs(t *testing.T) {
	fmt.Printf("Testing TestListAllNICs\n")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})
	assert.Nil(t, err)

	result, err := vmc.GetAllNICs(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, result)

}

func TestCreateAndDeleteNIC(t *testing.T) {
	// tests:
	// TODO: adding NIC with duplicate name should fail
	// TOTO: adding NIC with unique name should pass

	time_ms := time.Now().UnixNano()/1000000
	vmName := fmt.Sprintf("uvm-gotest_%d", time_ms)
	fmt.Printf("Testing TestCreateNIC with VM name = %s\n", vmName)

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})
	assert.Nil(t, err)

	vmConfig := &cloudyvm.VirtualMachineConfiguration{
		ID:   "VMid-" + vmName,
		Name: "VMname-" + vmName,
		Size: &cloudyvm.VmSize{
			Name: "Standard_DS1_v2",
		},
		SizeRequest: &cloudyvm.VmSizeRequest{
			SpecificSize: "Standard_DS1_v2",
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

	// Test subnet
	// subnet, err := vmc.FindBestSubnet(ctx, []string{"go-on-aws-vmSubnet"})
	// assert.Nil(t, err)
	// assert.NotEqual(t, "", subnet)

	// assert.NotNil(t, vmConfig.Size)

	// TODO: should there be a permanent dedicated subnet for testing, or should the test create/delete a subnet? 
	subnet := os.Getenv("TEST-SUBNET") // TODO: should there 

	err = vmc.CreateNIC(ctx, vmConfig, subnet)
	assert.Nil(t, err)
	assert.NotNil(t, vmConfig.PrimaryNetwork)
	assert.NotNil(t, vmConfig.PrimaryNetwork.ID)
	assert.NotNil(t, vmConfig.PrimaryNetwork.Name)
	assert.NotNil(t, vmConfig.PrimaryNetwork.PrivateIP)

	// verify the new NIC can be found by ID and matches created NIC
	createdNIC, err := vmc.FindNicByID(ctx, vmConfig.PrimaryNetwork.ID)
	assert.Nil(t, err)
	assert.NotNil(t, createdNIC)
	assert.Equal(t, true, assert.ObjectsAreEqualValues(vmConfig.PrimaryNetwork, createdNIC))

	// verify the new NIC can be found by Name and matches created NIC
	createdNICs, err := vmc.FindNICsByName(ctx, vmConfig.PrimaryNetwork.Name)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(createdNICs))
	assert.Equal(t, true, assert.ObjectsAreEqualValues(vmConfig.PrimaryNetwork, createdNICs[0]))

	// // manually modify nic in console, then confirm it fails equality test..
	// fmt.Printf("pausing for 60 seconds....")
	// time.Sleep(time.Duration(60)*time.Second)

	// // verify the modified NIC fails to match the originally created NIC
	// foundNIC, err = vmc.FindNicByID(ctx, vmConfig.PrimaryNetwork.ID)
	// assert.Nil(t, err)
	// assert.NotNil(t, foundNIC)
	// assert.Equal(t, true, assert.ObjectsAreEqualValues(vmConfig.PrimaryNetwork, foundNIC))

	// test NIC deletion
	nic_cache := vmConfig.PrimaryNetwork
	err = vmc.DeleteNIC(ctx, vmConfig)
	assert.Nil(t, err)
	assert.Nil(t, vmConfig.PrimaryNetwork)

	// confirm NIC ID is not found
	remainingNIC, err := vmc.FindNicByID(ctx, nic_cache.ID)
	assert.Nil(t, err)
	assert.Nil(t, remainingNIC)

	// confirm NIC Name is not found
	remainingNICs, err := vmc.FindNICsByName(ctx, nic_cache.Name)
	assert.Nil(t, err)
	assert.Nil(t, remainingNICs)
}
