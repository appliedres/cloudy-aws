package cloudyaws

import (
	"fmt"
	"os"
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"

	"github.com/stretchr/testify/assert"
)


func TestGetAvailableIPs(t *testing.T) {
	fmt.Printf("Testing TestGetAvailableIPs\n")

	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("test.env")

	vmc, err := NewAwsEc2Controller(ctx, &AwsEc2ControllerConfig{})
	assert.Nil(t, err)

	// TODO: should there be a permanent dedicated subnet for testing, or should the test create/delete a subnet? 
	subnetID := os.Getenv("TEST_SUBNET")

	numIPs, err := vmc.GetAvailableIPs(ctx, subnetID)
	assert.Nil(t, err)
	fmt.Printf("numIPs=%d\n", numIPs)
}