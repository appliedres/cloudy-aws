package cloudyaws

import (
	"fmt"
	"testing"

	"github.com/appliedres/cloudy"
	"github.com/appliedres/cloudy/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSecrets(t *testing.T) {
	fmt.Println("Testing AWS Secrets API")

	testSecretName := "go-test-secret"
	testSecretRawValue := "go-test-secret-value"
	testSecretBinaryValue := []byte(testSecretRawValue)
	
	ctx := cloudy.StartContext()

	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

	creds := AwsCredentials{
		Region:       cloudy.ForceEnv("AWS_REGION", ""),
		AccessKeyID:     cloudy.ForceEnv("AWS_ACCESS_KEY_ID", ""),
		SecretAccessKey:     cloudy.ForceEnv("AWS_SECRET_ACCESS_KEY", ""),
		SessionToken: cloudy.ForceEnv("AWS_SESSION_TOKEN", ""),
	}

	sm, err := NewSecretManager(ctx, creds)
	
	// start with a known state (deleted), will error if secret is already delted, but that's fine
	err = sm.DeleteSecret(ctx, testSecretName)

	// delete while non-existing, this should not produce an error
	err = sm.DeleteSecret(ctx, testSecretName)
	assert.Nil(t, err)

	// get raw secret that doesn't exist, should produce an error
	secRaw, err := sm.GetSecret(ctx, testSecretName)
	assert.NotNil(t, err)
	assert.Equal(t, secRaw, "")

	// get binary secret that doesn't exist, should produce an error
	secBin, err := sm.GetSecretBinary(ctx, testSecretName)
	assert.NotNil(t, err)
	assert.Nil(t, secBin)

	// save raw while non-existing
	err = sm.SaveSecret(ctx, testSecretName, testSecretRawValue)
	assert.Nil(t, err)

	// now exists, so get existing raw secret
	secRaw, err = sm.GetSecret(ctx, testSecretName)
	assert.Nil(t, err)
	assert.Equal(t, secRaw, testSecretRawValue)

	// Test overwriting raw value
	testSecretRawValue = "go-test-secret-value-updated"
	err = sm.SaveSecret(ctx, testSecretName, testSecretRawValue)
	assert.Nil(t, err)

	// Get overwritten raw value
	secRaw, err = sm.GetSecret(ctx, testSecretName)
	assert.Nil(t, err)
	assert.Equal(t, secRaw, testSecretRawValue)

	// save binary while secret exists
	err = sm.SaveSecretBinary(ctx, testSecretName, testSecretBinaryValue)
	assert.Nil(t, err)

	// get binary while secret exists
	secBin, err = sm.GetSecretBinary(ctx, testSecretName)
	assert.Nil(t, err)
	assert.Equal(t, secBin, testSecretBinaryValue)

	// delete while secret exists
	err = sm.DeleteSecret(ctx, testSecretName)
	assert.Nil(t, err)

	// confirm delete worked
	secRaw, err = sm.GetSecret(ctx, testSecretName)
	assert.NotNil(t, err)

	// TODO: can't create/get secret immediately after deletion due to AWS async deletion

	// // confirm delete worked
	// secBin, err = sm.GetSecretBinary(ctx, testSecretName)
	// assert.NotNil(t, err)

	// // save binary while non-existing
	// err = sm.SaveSecretBinary(ctx, testSecretName, testSecretBinaryValue)
	// assert.Nil(t, err)

	// // get binary while secret exists
	// secBin, err = sm.GetSecretBinary(ctx, testSecretName)
	// assert.Nil(t, err)
	// assert.Equal(t, secBin, testSecretBinaryValue)

	// // final delete to clean up
	// err = sm.DeleteSecret(ctx, testSecretName)
	// assert.Nil(t, err)

}

// // passes if one or more secrets is listed
// func ListAllSecrets(t *testing.T) {
// 	fmt.Println("Test: SecretManager - ListAll")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")
	
// 	all, err := sm.ListAll(ctx)
// 	assert.Nil(t, err)
// 	assert.NotNil(t, all)
// }

// func SaveSecret(t *testing.T) {
// 	fmt.Println("Test: SecretManager - Save")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")

// 	err = sm.SaveSecret(ctx, testSecretName, testSecretRawValue)
// 	assert.Nil(t, err)
// }

// func SaveSecretBinary(t *testing.T) {
// 	fmt.Println("Test: SecretManager - Save")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")

// 	err = sm.SaveSecretBinary(ctx, testSecretName, testSecretBinaryValue)
// 	assert.Nil(t, err)
// }

// func GetSecret(t *testing.T) {
// 	fmt.Println("Test: SecretManager - Save")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")

// 	sv, err := sm.GetSecret(ctx, testSecretName)
// 	assert.Nil(t, err)
// 	assert.Equal(t, sv, testSecretRawValue)
// }

// func GetSecretBinary(t *testing.T) {
// 	fmt.Println("Test: SecretManager - Save")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")

// 	sv, err := sm.GetSecretBinary(ctx, testSecretName)
// 	assert.Nil(t, err)
// 	assert.Equal(t, sv, testSecretBinaryValue)
// }

// func DeleteSecret(ctx, t *testing.T) {
// 	fmt.Println("Test: SecretManager - Delete")

// 	ctx := cloudy.StartContext()

// 	_ = testutil.LoadEnv("../arkloud-conf/arkloud.env")

// 	sm, err := NewSecretManager(ctx, "us-east-1")

// 	err = sm.DeleteSecret(ctx, ctx, testSecretName)
// 	assert.Nil(t, err)
// }